// Package remote posts the note listing to a webhook. It is the program's
// only outbound network call, and therefore the only place where a hung peer
// could stall the process and where a credential could reach a log line.
package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// requestTimeout bounds the whole exchange — connect, TLS, request,
	// response headers and body. Transport timeouts each bound one phase, so
	// all of them can pass while the call still hangs.
	requestTimeout = 10 * time.Second

	// maxResponseBytes caps what we read back. The peer is untrusted; without
	// a cap it decides how much memory we use.
	maxResponseBytes = 64 << 10
)

// Errors callers match with errors.Is.
var (
	ErrInvalidURL = errors.New("webhook url is not usable")
	ErrStatus     = errors.New("webhook rejected the request")
)

// NewClient returns the client used for every outbound call. The timeout is
// in the literal so a reader can see the bound without following a variable.
func NewClient() *http.Client {
	return &http.Client{Timeout: requestTimeout}
}

// Publish POSTs body to rawURL. The caller's context bounds the call: if the
// program is shutting down, the request is cancelled with it.
//
// The URL is parsed here rather than inside net/http so that a bad one fails
// with an error of ours. The standard library's parse error quotes the whole
// URL, and a webhook URL is a secret.
func Publish(ctx context.Context, client *http.Client, rawURL string, body []byte) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: it does not parse", ErrInvalidURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: %w: scheme %q is not http(s)", Redact(rawURL), ErrInvalidURL, u.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request for %s: %w", Redact(rawURL), ErrInvalidURL)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post %s: %w", Redact(rawURL), err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("post %s: %w: %s", Redact(rawURL), ErrStatus, resp.Status)
	}
	return nil
}

// Redact reduces a URL to the parts that are safe to log: scheme and host.
// Webhook URLs are secrets, and the token can be anywhere in them — the query
// string, the userinfo, or (Slack, Discord, and most others) a path segment,
// which is why the path goes too. An error message containing one is a secret
// written to every log that catches it, so redact more than you think you need.
func Redact(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "(unparsable url)"
	}
	u.User = nil
	if u.Path != "" || u.RawPath != "" {
		u.Path = "/redacted"
		u.RawPath = ""
	}
	if u.RawQuery != "" {
		u.RawQuery = "redacted"
	}
	u.Fragment = ""
	return u.String()
}
