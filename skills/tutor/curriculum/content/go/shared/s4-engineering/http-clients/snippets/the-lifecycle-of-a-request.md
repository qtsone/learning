In Go:

```go
resp, err := client.Do(req)
if err != nil {
	return err // transport failure: DNS, refused, timeout — there is no response
}
defer resp.Body.Close() // ALWAYS, on every path, before touching the status
```

Note the split: `err != nil` means *no HTTP conversation happened*. A 404 is
a successful conversation with a disappointing answer — `err` is nil and
`resp.StatusCode` is 404. Confusing these two is the classic first bug in
every Go HTTP client.
