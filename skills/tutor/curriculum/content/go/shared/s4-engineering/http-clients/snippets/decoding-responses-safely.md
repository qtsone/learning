In Go:

```go
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error: status %d", e.StatusCode)
}
```

Callers use `errors.As(err, &apiErr)` to fish it out of a wrapped chain, then
branch on `apiErr.StatusCode`. For the happy path, decode straight from the
body stream (`json.NewDecoder(resp.Body).Decode(&v)` in Go) into the struct
you defined — the JSON techniques from S3 apply unchanged.
