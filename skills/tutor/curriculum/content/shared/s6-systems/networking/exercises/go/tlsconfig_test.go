package netlab

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"testing"
)

// dialTLS runs a full handshake over an in-memory pipe and returns both ends.
// The returned error is the client's: that is the side that verifies, and the
// side whose failures this lesson cares about.
func dialTLS(t *testing.T, clientCfg, serverCfg *tls.Config) (client, server *tls.Conn, err error) {
	t.Helper()
	rawClient, rawServer := netPipe(t)

	server = tls.Server(rawServer, serverCfg)
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Handshake() }()

	client = tls.Client(rawClient, clientCfg)
	if err := client.Handshake(); err != nil {
		rawClient.Close() // unblock the server half, then drain it
		<-serverErr
		return nil, nil, err
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("client handshake succeeded but the server's failed: %v", err)
	}
	return client, server, nil
}

func TestClientTLSConfig(t *testing.T) {
	roots, _ := newTestCA(t, testServerName)
	cfg := ClientTLSConfig(roots, testServerName)
	if cfg == nil {
		t.Fatal("ClientTLSConfig returned nil")
	}
	if cfg.RootCAs != roots {
		t.Fatal("ClientTLSConfig did not set RootCAs to the pool it was given; a nil RootCAs means 'trust the machine's store', which is not what an internal service should accept")
	}
	if cfg.ServerName != testServerName {
		t.Fatalf("ServerName = %q; want %q — without it the certificate is never checked against the host you meant to reach", cfg.ServerName, testServerName)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify is true: the connection is still encrypted, but to whoever answered")
	}
	if cfg.MinVersion < tls.VersionTLS12 {
		t.Fatalf("MinVersion = %#04x; want at least TLS 1.2 (%#04x) — leaving it zero permits obsolete versions", cfg.MinVersion, tls.VersionTLS12)
	}
}

func TestServerTLSConfig(t *testing.T) {
	_, cert := newTestCA(t, testServerName)
	cfg := ServerTLSConfig(cert)
	if cfg == nil {
		t.Fatal("ServerTLSConfig returned nil")
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("len(Certificates) = %d; want 1 — the server has nothing to present otherwise", len(cfg.Certificates))
	}
	if !bytes.Equal(cfg.Certificates[0].Certificate[0], cert.Certificate[0]) {
		t.Fatal("ServerTLSConfig presents a different certificate than the one it was given")
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify has no business being set here")
	}
	if cfg.MinVersion < tls.VersionTLS12 {
		t.Fatalf("MinVersion = %#04x; want at least TLS 1.2 (%#04x)", cfg.MinVersion, tls.VersionTLS12)
	}
}

func TestHandshakeSucceedsAndEchoesFrames(t *testing.T) {
	roots, cert := newTestCA(t, testServerName)
	client, server, err := dialTLS(t, ClientTLSConfig(roots, testServerName), ServerTLSConfig(cert))
	if err != nil {
		t.Fatalf("handshake with a trusted CA and the right name = %v; want it to succeed", err)
	}
	if v := client.ConnectionState().Version; v < tls.VersionTLS12 {
		t.Fatalf("negotiated version %#04x; want TLS 1.2 or newer", v)
	}

	// A *tls.Conn is just another net.Conn: the framing code above runs over
	// it unchanged.
	done := make(chan error, 1)
	go func() { done <- ServeEcho(server) }()

	payload := []byte("frames over TLS")
	if err := WriteFrame(client, payload); err != nil {
		t.Fatalf("WriteFrame over TLS = %v; want nil", err)
	}
	got, err := ReadFrame(client)
	if err != nil {
		t.Fatalf("ReadFrame over TLS = %v; want %q back", err, payload)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("echo over TLS returned %q; want %q", got, payload)
	}

	client.Close()
	if err := <-done; err != nil {
		t.Fatalf("ServeEcho over TLS after a clean close = %v; want nil", err)
	}
}

func TestHandshakeFailsOnWrongServerName(t *testing.T) {
	roots, cert := newTestCA(t, testServerName)
	_, _, err := dialTLS(t, ClientTLSConfig(roots, "evil.test"), ServerTLSConfig(cert))
	if err == nil {
		t.Fatal("handshake succeeded against a certificate issued for another name; the name check is what makes the chain mean anything")
	}
	var nameErr x509.HostnameError
	if !errors.As(err, &nameErr) {
		t.Fatalf("handshake error = %v; want an x509.HostnameError (the certificate is valid, just not for this host)", err)
	}
}

func TestHandshakeFailsOnUntrustedCA(t *testing.T) {
	_, cert := newTestCA(t, testServerName)
	empty := x509.NewCertPool() // trusts nobody
	_, _, err := dialTLS(t, ClientTLSConfig(empty, testServerName), ServerTLSConfig(cert))
	if err == nil {
		t.Fatal("handshake succeeded against a certificate from an untrusted authority")
	}
	var authErr x509.UnknownAuthorityError
	if !errors.As(err, &authErr) {
		t.Fatalf("handshake error = %v; want an x509.UnknownAuthorityError — the chain must reach a root in the pool you were handed", err)
	}
}
