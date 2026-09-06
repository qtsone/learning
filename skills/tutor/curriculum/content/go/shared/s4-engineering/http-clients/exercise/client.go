// Package client is a small JSON API client built the way production
// clients are: explicit timeouts, closed bodies, typed errors, and a
// retry policy you can defend.
package client

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// maxErrorBody caps how much of a non-2xx response body is kept in an
// APIError — error bodies are someone else's output, never trust their size.
const maxErrorBody = 4 * 1024

type Client struct {
	BaseURL string
	HTTP    *http.Client

	// sleep is called for retry waits. Tests replace it to observe the
	// waits without actually sleeping — use it instead of time.Sleep.
	sleep func(time.Duration)
}

// New returns a Client for the API at baseURL.
func New(baseURL string) *Client {
	// TODO: give the http.Client a 10-second overall timeout.
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{},
		sleep:   time.Sleep,
	}
}

// APIError is a non-2xx answer from the server: the HTTP conversation
// succeeded, but the server said no.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	// TODO: mention the status code, e.g. "api error: status 503".
	return "TODO"
}

// GetJSON issues a GET for BaseURL+path and decodes the JSON response
// into v. Non-2xx responses become an *APIError.
func (c *Client) GetJSON(ctx context.Context, path string, v any) error {
	// TODO:
	//   1. build the request with http.NewRequestWithContext
	//   2. set the Accept header to "application/json"
	//   3. send it via c.HTTP
	//   4. close the body on every path
	//   5. non-2xx: return *APIError with at most maxErrorBody bytes of body
	//   6. 2xx: decode JSON into v
	return errors.New("TODO: implement GetJSON")
}
