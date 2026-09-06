package netlab

import (
	"crypto/tls"
	"crypto/x509"
)

// MinTLSVersion is the floor both sides refuse to go below. A zero MinVersion
// lets the library pick, and "whatever the peer suggests" is not a security
// decision you want made for you.
const MinTLSVersion = tls.VersionTLS12

// ClientTLSConfig returns the client half of the connection: trust exactly
// the certificate authorities in roots, and require the peer's certificate to
// be valid for serverName.
//
// Both halves matter. A verified chain proves the certificate is genuine; the
// name check proves it is genuine *for the host you meant to reach*. Skip the
// second and a valid certificate for evil.example passes for api.example.
func ClientTLSConfig(roots *x509.CertPool, serverName string) *tls.Config {
	return &tls.Config{
		RootCAs:    roots,
		ServerName: serverName,
		MinVersion: MinTLSVersion,
	}
}

// ServerTLSConfig returns the server half: present cert to clients, with the
// same version floor.
func ServerTLSConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   MinTLSVersion,
	}
}
