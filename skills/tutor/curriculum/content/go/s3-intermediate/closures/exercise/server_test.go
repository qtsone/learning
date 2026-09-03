package closures

import "testing"

func TestNewServerDefaults(t *testing.T) {
	s := NewServer()
	if s.host != "localhost" {
		t.Errorf("default host = %q, want %q", s.host, "localhost")
	}
	if s.port != 8080 {
		t.Errorf("default port = %d, want %d", s.port, 8080)
	}
	if s.banner != "ready" {
		t.Errorf("default banner = %q, want %q", s.banner, "ready")
	}
}

func TestNewServerOptionsOverrideDefaults(t *testing.T) {
	s := NewServer(WithHost("example.internal"), WithPort(9090), WithBanner("hello"))
	if s.host != "example.internal" {
		t.Errorf("host = %q, want %q", s.host, "example.internal")
	}
	if s.port != 9090 {
		t.Errorf("port = %d, want %d", s.port, 9090)
	}
	if s.banner != "hello" {
		t.Errorf("banner = %q, want %q", s.banner, "hello")
	}
}

func TestNewServerPartialOptionsKeepOtherDefaults(t *testing.T) {
	s := NewServer(WithPort(9090))
	if s.port != 9090 {
		t.Errorf("port = %d, want %d", s.port, 9090)
	}
	if s.host != "localhost" {
		t.Errorf("host = %q, want %q (unset knobs must keep their defaults)", s.host, "localhost")
	}
	if s.banner != "ready" {
		t.Errorf("banner = %q, want %q (unset knobs must keep their defaults)", s.banner, "ready")
	}
}

func TestNewServerLastOptionWins(t *testing.T) {
	s := NewServer(WithPort(1000), WithPort(2000))
	if s.port != 2000 {
		t.Errorf("port = %d, want 2000 (options must apply in order, later ones winning)", s.port)
	}
}
