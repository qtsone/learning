In Go this is `net/http/httptest`:

```go
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, `{"name":"go"}`)
}))
defer ts.Close()
// ts.URL is the server's address — point your client at it
```
