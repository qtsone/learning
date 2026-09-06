In Go: `crypto/tls` performs the whole verification by default — you
configure *what to trust* and *what to require*, and never turn checks off:

```go
// Client: which roots to trust, which name to demand, floor the version.
cfg := &tls.Config{
    RootCAs:    pool,             // nil means the system trust store
    ServerName: "api.internal",   // the name the certificate must match
    MinVersion: tls.VersionTLS13,
}

// Server: present a certificate, floor the version.
cfg := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS13,
}
```

Which floor? TLS 1.3 is the one to demand when you control both ends — it
drops the legacy ciphers and cuts a round trip from the handshake. TLS 1.2
is the floor you pick when you must still reach peers you don't control;
everything below it is broken and non-negotiable. The exercise floors at
1.2 for exactly that reason, and the rule to carry away is *set the floor
deliberately* — a zero `MinVersion` means the library, not you, decided.
