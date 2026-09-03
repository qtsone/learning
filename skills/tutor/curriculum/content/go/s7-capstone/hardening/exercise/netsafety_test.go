package capstone

import "testing"

// Criterion 6: every outbound HTTP client is bounded. A client with no
// Timeout is a promise to wait forever, and forever is the one deadline you
// never chose deliberately.
func TestHTTPClientsHaveTimeouts(t *testing.T) {
	dir := project(t)
	findings, err := scanHTTPClients(dir)
	if err != nil {
		t.Fatalf("scanning %s: %v", dir, err)
	}
	if len(findings) == 0 {
		return
	}
	for _, f := range findings {
		t.Errorf("%s", f)
	}
	t.Log(`Set the timeout where a reviewer can see it — in the literal:

    var client = &http.Client{Timeout: 10 * time.Second}

Client.Timeout bounds the whole exchange: connect, TLS handshake, request
write, response headers, and reading the body. Transport-level timeouts each
bound one phase, so they can all pass while the call still hangs, which is why
this check wants the field on the client itself. http.DefaultClient and the
http.Get/Post helpers that use it have no timeout at all.

If a call genuinely has to be unbounded, give it an explicit
context.WithTimeout at the call site and say so in your security document —
"unbounded" written down is a decision; "unbounded" by omission is a hang.`)
}

// Criterion 7: context discipline in non-test code. Two rules, both
// mechanically decidable: no context.TODO() anywhere, and no
// context.Background() inside a function that was handed a context.
func TestContextIsPropagated(t *testing.T) {
	dir := project(t)
	findings, err := scanContextUse(dir)
	if err != nil {
		t.Fatalf("scanning %s: %v", dir, err)
	}
	if len(findings) == 0 {
		return
	}
	for _, f := range findings {
		t.Errorf("%s", f)
	}
	t.Log(`A context you were given carries a deadline, a cancellation signal, and
whatever request-scoped values sit above you. Calling context.Background()
inside a function that already has one throws all three away: the caller hangs
up, and your work carries on against a database that nobody is waiting for.
Pass ctx down instead.

If you deliberately want work to outlive the request — flushing a buffer,
finishing an audit write — that is context.WithoutCancel(ctx), which keeps the
values and drops the cancellation, plus a timeout of its own. Write it that way
and the intent is legible.

Roots belong in main (or a run function main calls), where the program's
lifetime actually starts. context.TODO() belongs nowhere in finished code:
it is a marker saying someone still has to decide which context goes here.`)
}
