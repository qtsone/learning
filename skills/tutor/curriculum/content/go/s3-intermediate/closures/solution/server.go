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
	return func(s *Server) { s.host = host }
}

// WithPort returns an Option that sets the server's port.
func WithPort(port int) Option {
	return func(s *Server) { s.port = port }
}

// WithBanner returns an Option that sets the server's banner.
func WithBanner(banner string) Option {
	return func(s *Server) { s.banner = banner }
}

// NewServer builds a Server with defaults host "localhost", port 8080 and
// banner "ready", then applies opts in order — later options win.
func NewServer(opts ...Option) *Server {
	s := &Server{
		host:   "localhost",
		port:   8080,
		banner: "ready",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
