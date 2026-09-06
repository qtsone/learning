**In Go:** one goroutine per connection with a buffered outbound channel,
where the `default` case is the whole point — a slow client must never block
the fan-out serving everyone else:

```go
func (g *Gateway) deliver(userID string, msg []byte) {
	g.mu.RLock()
	sessions := g.sessions[userID] // one entry per connected device
	g.mu.RUnlock()

	for _, s := range sessions {
		select {
		case s.out <- msg:
		default:
			s.close() // the client resyncs from its cursor on reconnect
		}
	}
}
```

Dropping a laggard is safe *only* because the durable log is the source of
truth and the client can replay from its cursor — the same reasoning as
your S3 worker-pool backpressure, applied to a socket.
