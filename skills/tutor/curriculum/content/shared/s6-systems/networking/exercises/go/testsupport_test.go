package netlab

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// testServerName is the only name the test certificate is valid for.
const testServerName = "netlab.test"

// netPipe returns the two ends of an in-memory, synchronous net.Conn pair.
// No sockets, no ports, nothing leaves the process — but the Read/Write
// semantics are a TCP connection's, partial reads included.
//
// The deadline is a deadlock guard: a stuck implementation fails with a
// message instead of hanging the suite. It is not a timing assertion, and a
// correct implementation never comes near it.
func netPipe(t *testing.T) (client, server net.Conn) {
	t.Helper()
	c, s := net.Pipe()
	deadline := time.Now().Add(10 * time.Second)
	if err := c.SetDeadline(deadline); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	if err := s.SetDeadline(deadline); err != nil {
		t.Fatalf("set server deadline: %v", err)
	}
	t.Cleanup(func() {
		c.Close()
		s.Close()
	})
	return c, s
}

// newTestCA mints a throwaway certificate authority and a leaf certificate
// valid for names, entirely in memory. This is the tests' own trust anchor:
// the machine's trust store is never consulted and no real CA exists.
func newTestCA(t *testing.T, names ...string) (*x509.CertPool, tls.Certificate) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "netlab test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("self-sign CA certificate: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: names[0]},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     names,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("sign leaf certificate: %v", err)
	}
	return roots, tls.Certificate{
		Certificate: [][]byte{leafDER},
		PrivateKey:  leafKey,
	}
}
