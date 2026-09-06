# Your PR — "client: add FetchTodo"

Now you are the author. Last week you opened a small PR adding one method
to the API client you built in the HTTP-clients lesson. This is the new
code:

```go
// FetchTodo fetches the todo with the given id from the API.
func (c *Client) FetchTodo(ctx context.Context, id int) (Todo, error) {
	url := fmt.Sprintf("%s/todos/%d", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Todo{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Todo{}, fmt.Errorf("fetch todo %d: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Todo{}, fmt.Errorf("fetch todo %d: unexpected status %s", id, resp.Status)
	}
	var t Todo
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return Todo{}, fmt.Errorf("decode todo %d: %w", id, err)
	}
	return t, nil
}
```

The reviewer left four comments. In `NOTES.md`, write the reply you would
post to each — decide whether to concede, push back, or answer, and give
your reasoning either way.

---

**Comment A** *(blocking)*:

> A missing todo (404) and a network failure come back as the same opaque
> error string. Callers will need to treat "not found" differently from
> "the API is down" — can we give them a sentinel, e.g. `ErrNotFound`,
> for the 404 case? `errors.Is` support is the whole point of our error
> wrapping.

**Comment B** *(suggestion)*:

> `t` is too cryptic — rename it to `todoResult`. Also please add a
> `// decode the response body` comment above the Decode call so readers
> can follow along.

**Comment C** *(suggestion)*:

> The API flakes sometimes. Could this method retry with backoff a couple
> of times before giving up?

**Comment D** *(question)*:

> Why `json.NewDecoder(resp.Body)` instead of reading the whole body with
> `io.ReadAll` and calling `json.Unmarshal`? Genuinely asking — I've seen
> both in our codebase.
