package remote_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tutor.local/capstone-reference/internal/remote"
)

// A client with no timeout is the defect this test exists to prevent coming
// back, so it asserts on the property rather than on the constant.
func TestNewClientHasTimeout(t *testing.T) {
	if got := remote.NewClient().Timeout; got <= 0 {
		t.Fatalf("NewClient().Timeout = %v, want a positive bound", got)
	}
}

func TestRedactRemovesCredentials(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"query token", "https://hooks.example/x?token=s3cret", "https://hooks.example/redacted?redacted"},
		{"userinfo", "https://user:pw@hooks.example/x", "https://hooks.example/redacted"},
		{"fragment", "https://hooks.example/x#anchor", "https://hooks.example/redacted"},
		{"token in the path", "https://hooks.example/services/T00/B00/s3cret", "https://hooks.example/redacted"},
		{"nothing but a host", "https://hooks.example", "https://hooks.example"},
		{"unparsable", "://nope", "(unparsable url)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := remote.Redact(c.in); got != c.want {
				t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// Both failure paths are reachable without a network: a URL that does not
// parse, and one that parses but is not http(s). Nothing is dialled.
func TestPublishRejectsBadURLs(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"unparsable", "://nope"},
		{"wrong scheme", "ftp://hooks.example/x?token=s3cret"},
		{"file scheme", "file:///etc/passwd"},
		{"empty", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := remote.Publish(context.Background(), remote.NewClient(), c.url, []byte("body"))
			if !errors.Is(err, remote.ErrInvalidURL) {
				t.Fatalf("Publish(%q) error = %v, want ErrInvalidURL", c.url, err)
			}
			if strings.Contains(err.Error(), "s3cret") {
				t.Errorf("error text leaked the token: %v", err)
			}
		})
	}
}
