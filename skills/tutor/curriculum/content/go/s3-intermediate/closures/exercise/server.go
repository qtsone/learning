package closures

// Server holds configuration assembled by NewServer. The fields stay
// unexported: callers configure a Server only through options.
type Server struct {
	host   string
	port   int
	banner string
}

// Option configures a Server under construction.
type Option func(*Server)

// WithHost returns an Option that sets the server's host.
func WithHost(host string) Option {
	// TODO: return a closure that sets s.host.
	return func(s *Server) {}
}

// WithPort returns an Option that sets the server's port.
func WithPort(port int) Option {
	// TODO: return a closure that sets s.port.
	return func(s *Server) {}
}

// WithBanner returns an Option that sets the server's banner.
func WithBanner(banner string) Option {
	// TODO: return a closure that sets s.banner.
	return func(s *Server) {}
}

// NewServer builds a Server with defaults host "localhost", port 8080 and
// banner "ready", then applies opts in order — later options win.
func NewServer(opts ...Option) *Server {
	// TODO: set the defaults, then apply each option to the server.
	return &Server{}
}
