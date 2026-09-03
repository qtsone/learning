package board

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"
)

// Page limits. A caller-supplied limit is clamped rather than refused: an
// unbounded page is a caller deciding how much memory you spend.
const (
	DefaultPageLimit = 20
	MaxPageLimit     = 50
)

// Config wires the service. Every zero value has a defensible default except
// Store, which the service cannot invent.
type Config struct {
	Store    *Store
	Clock    Clock
	Hasher   PasswordHasher
	Hub      *Hub
	Policy   *Policy
	Sessions *SessionStore
	Limiter  *Limiter
	Logger   *slog.Logger

	// NewID mints task ids. Injected so the tests get stable ids — and so the
	// production wiring can swap in whatever your ids are.
	NewID func() string

	// CookieSecure defaults to false so plain-HTTP local development works. Set
	// it true everywhere else.
	CookieSecure bool

	// Heartbeat is how often the event stream sends a comment frame to keep an
	// idle connection out of a proxy's reaper. Zero disables it.
	Heartbeat time.Duration

	// StreamRetry is the reconnection delay advertised to clients.
	StreamRetry time.Duration
}

// Server is the service. It owns no HTTP state of its own: everything a
// request needs is either injected here or carried on the request.
type Server struct {
	store    *Store
	clock    Clock
	hasher   PasswordHasher
	hub      *Hub
	policy   *Policy
	sessions *SessionStore
	limiter  *Limiter
	log      *slog.Logger

	newID        func() string
	cookieSecure bool
	heartbeat    time.Duration
	streamRetry  time.Duration

	// dummyHash exists so an unknown username still costs one verification. It
	// is not a secret; its job is to make a failed login take the same work
	// whether the account exists or not, so latency is not an account-discovery
	// oracle.
	dummyHash string

	seq atomic.Int64
}

const dummyPassword = "there-is-no-account-with-this-password"

// NewServer assembles the service from cfg.
func NewServer(cfg Config) *Server {
	clock := cfg.Clock
	if clock == nil {
		clock = RealClock{}
	}
	hasher := cfg.Hasher
	if hasher == nil {
		hasher = BcryptHasher{}
	}
	hub := cfg.Hub
	if hub == nil {
		hub = NewHub(16, 64, cfg.Logger)
	}
	policy := cfg.Policy
	if policy == nil {
		policy = NewPolicy(DefaultRules, cfg.Logger)
	}
	sessions := cfg.Sessions
	if sessions == nil {
		sessions = NewSessionStore(clock, DefaultSessionTTL)
	}
	limiter := cfg.Limiter
	if limiter == nil {
		limiter = NewLimiter(10, 20, clock)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	dummyHash, _ := hasher.Hash(dummyPassword)

	s := &Server{
		store:        cfg.Store,
		clock:        clock,
		hasher:       hasher,
		hub:          hub,
		policy:       policy,
		sessions:     sessions,
		limiter:      limiter,
		log:          logger,
		newID:        cfg.NewID,
		cookieSecure: cfg.CookieSecure,
		heartbeat:    cfg.Heartbeat,
		streamRetry:  cfg.StreamRetry,
		dummyHash:    dummyHash,
	}
	if s.newID == nil {
		s.newID = s.sequentialID
	}
	return s
}

// Sweep drops state that has aged out and reports what it removed: rate-limit
// buckets nobody has touched for long enough to have refilled anyway, and
// sessions past their expiry.
//
// Both maps are keyed on something a caller controls — an account id or an
// address — so neither may grow forever. The limiter's is the sharper edge:
// anonymous callers key on the peer address, and anyone holding a /64 of IPv6
// can make you allocate a bucket per request until the process dies. Evicting
// no sooner than RefillWindow is what keeps the eviction from being a way
// around the limit.
func (s *Server) Sweep() (buckets, sessions int) {
	return s.limiter.Cleanup(s.limiter.RefillWindow()), s.sessions.Cleanup()
}

// StartJanitor runs Sweep on every tick of a ticker from the injected Clock and
// returns when ctx is done. It blocks, so main runs it on its own goroutine and
// cancels ctx during shutdown.
//
// Nothing else in this service removes an entry from either map. A service that
// never starts this is the api-hardening lesson's memory-exhaustion vector with
// a login endpoint attached.
func (s *Server) StartJanitor(ctx context.Context, every time.Duration) {
	ticks, stop := s.clock.NewTicker(every)
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			buckets, sessions := s.Sweep()
			s.log.Debug("janitor swept", "buckets", buckets, "sessions", sessions)
		}
	}
}

// Hub exposes the fan-out for wiring — the worker pool publishes into it too.
func (s *Server) Hub() *Hub { return s.hub }

// Sessions exposes the session store for wiring and tests.
func (s *Server) Sessions() *SessionStore { return s.sessions }

// Policy exposes the access-control policy for wiring and tests.
func (s *Server) Policy() *Policy { return s.policy }

func (s *Server) sequentialID() string {
	return "t-" + strconv.FormatInt(s.seq.Add(1), 10)
}
