// Package client is a small JSON API client built the way production
// clients are: explicit timeouts, closed bodies, typed errors, and a
// retry policy you can defend.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
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
	return fmt.Sprintf("api error: status %d", e.StatusCode)
}

// GetJSON issues a GET for BaseURL+path and decodes the JSON response
// into v. Non-2xx responses become an *APIError.
func (c *Client) GetJSON(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
