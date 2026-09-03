package gql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// ------------------------------------------------------------ N+1 and the fix

// The hazard, measured. Five posts, and the naive Post.author resolver runs
// once per post: 1 query for the list and N for the authors.
func TestNaiveResolversAreNPlusOne(t *testing.T) {
	store := NewStore()
	ts := startServer(t, store, NaiveSchema(store))

	status, body := postQuery(t, ts, qPostsWithAuthors)
	requireStatus(t, status, http.StatusOK)
	if n := len(at(t, body, "data", "posts").([]any)); n != 5 {
		t.Fatalf("got %d posts, want 5", n)
	}

	requireCallCount(t, store, "Posts", 1)
	requireCallCount(t, store, "AuthorByID", 5)
	if got := store.TotalCalls(); got != 6 {
		t.Errorf("total store calls = %d, want 6 (1 + N)", got)
	}
}

// The same query, the same schema shape, the same answer — through loaders.
func TestBatchedResolversMakeOneCallPerLevel(t *testing.T) {
	ts, store := batchedServer(t)

	status, body := postQuery(t, ts, qPostsWithAuthors)
	requireStatus(t, status, http.StatusOK)
	if got := at(t, body, "data", "posts", 0, "author", "name"); got != "Ada Lovelace" {
		t.Fatalf("posts[0].author.name = %v, want Ada Lovelace", got)
	}

	requireCallCount(t, store, "AuthorByID", 0)
	requireCallCount(t, store, "AuthorsByIDs", 1)
	calls := store.Calls("AuthorsByIDs")
	if len(calls) == 1 {
		want := []string{"a1", "a2", "ghost"}
		if !reflect.DeepEqual(calls[0].Keys, want) {
			t.Errorf("batched keys = %v, want %v: five posts, three distinct authors", calls[0].Keys, want)
		}
	}
	if got := store.TotalCalls(); got != 2 {
		t.Errorf("total store calls = %d, want 2 (the list, then one batch)", got)
	}
}

// Two levels of nesting, three store calls: the list, the authors, the
// comments. The comment authors cost nothing because the post level already
// loaded them into the same loader.
func TestNestedQueryReusesTheLoaderCache(t *testing.T) {
	ts, store := batchedServer(t)

	status, _ := postQuery(t, ts, qDeepTree)
	requireStatus(t, status, http.StatusOK)

	requireCallCount(t, store, "Posts", 1)
	requireCallCount(t, store, "AuthorsByIDs", 1)
	requireCallCount(t, store, "CommentsByPostIDs", 1)
	requireCallCount(t, store, "CommentsByPost", 0)
	if got := store.TotalCalls(); got != 3 {
		t.Errorf("total store calls = %d, want 3: the comment authors were already loaded", got)
	}
}

// Author.posts closes the cycle, and it batches like everything else.
func TestAuthorPostsBatches(t *testing.T) {
	ts, store := batchedServer(t)

	status, body := postQuery(t, ts, qAuthorPosts)
	requireStatus(t, status, http.StatusOK)
	if got := at(t, body, "data", "posts", 0, "author", "posts", 0, "title"); got != "Notes on the Analytical Engine" {
		t.Fatalf("posts[0].author.posts[0].title = %v", got)
	}

	requireCallCount(t, store, "PostsByAuthor", 0)
	requireCallCount(t, store, "PostsByAuthorIDs", 1)
	if got := store.TotalCalls(); got != 3 {
		t.Errorf("total store calls = %d, want 3", got)
	}
}

// Batching is an implementation detail: the client must not be able to tell.
func TestBatchedAndNaiveReturnIdenticalData(t *testing.T) {
	naiveStore := NewStore()
	naive := startServer(t, naiveStore, NaiveSchema(naiveStore))
	batched, _ := batchedServer(t)

	_, naiveBody := postQuery(t, naive, qDeepTree)
	_, batchedBody := postQuery(t, batched, qDeepTree)

	if !reflect.DeepEqual(naiveBody["data"], batchedBody["data"]) {
		t.Errorf("batching changed the response:\nnaive:   %v\nbatched: %v", naiveBody["data"], batchedBody["data"])
	}
}

// ------------------------------------------------------------ error semantics

// One field of one item fails. The rest of the tree is still returned, the
// status is still 200, and the errors array says exactly where it broke.
func TestPartialDataCarriesAnErrorPath(t *testing.T) {
	ts, _ := batchedServer(t)

	status, body := postQuery(t, ts, qPostsWithAuthors)
	requireStatus(t, status, http.StatusOK)

	if got := at(t, body, "data", "posts", 4, "title"); got != "The orphaned draft" {
		t.Errorf("posts[4].title = %v: the sibling field must still resolve", got)
	}
	if got := at(t, body, "data", "posts", 4, "author"); got != nil {
		t.Errorf("posts[4].author = %v, want null", got)
	}
	if got := at(t, body, "data", "posts", 0, "author", "name"); got != "Ada Lovelace" {
		t.Errorf("posts[0].author.name = %v: one bad row must not empty the response", got)
	}

	errs := responseErrors(t, body)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Path != "posts.4.author" {
		t.Errorf("error path = %q, want \"posts.4.author\"", errs[0].Path)
	}
	if !strings.Contains(errs[0].Message, "ghost") {
		t.Errorf("error message = %q, want it to name the missing author", errs[0].Message)
	}
}

// A nullable field with nothing to return is null, and that is not an error.
func TestMissingOptionalFieldIsNullWithoutAnError(t *testing.T) {
	ts, _ := batchedServer(t)

	status, body := postQuery(t, ts, `{ post(id: "nope") { title } }`)
	requireStatus(t, status, http.StatusOK)

	if got := at(t, body, "data", "post"); got != nil {
		t.Errorf("data.post = %v, want null", got)
	}
	if errs := responseErrors(t, body); len(errs) != 0 {
		t.Errorf("got errors %v, want none: null is a legitimate answer", errs)
	}
}

// ------------------------------------------------------------ mutations

func TestMutationWritesAndReturnsTheNewObject(t *testing.T) {
	ts, _ := batchedServer(t)

	status, body := postQuery(t, ts,
		`mutation { createPost(authorId: "a1", title: "Fresh ink") { id title author { name } } }`)
	requireStatus(t, status, http.StatusOK)

	if got := at(t, body, "data", "createPost", "title"); got != "Fresh ink" {
		t.Errorf("createPost.title = %v", got)
	}
	if got := at(t, body, "data", "createPost", "author", "name"); got != "Ada Lovelace" {
		t.Errorf("createPost.author.name = %v: a mutation's payload resolves like any other object", got)
	}

	_, after := postQuery(t, ts, `{ posts(first: 100) { id } }`)
	if n := len(at(t, after, "data", "posts").([]any)); n != 6 {
		t.Errorf("got %d posts after the mutation, want 6", n)
	}
}

func TestMutationFailureIsAFieldError(t *testing.T) {
	ts, _ := batchedServer(t)

	status, body := postQuery(t, ts,
		`mutation { createPost(authorId: "nobody", title: "X") { id } }`)
	requireStatus(t, status, http.StatusOK)

	if got := at(t, body, "data", "createPost"); got != nil {
		t.Errorf("data.createPost = %v, want null", got)
	}
	errs := responseErrors(t, body)
	if len(errs) != 1 || errs[0].Path != "createPost" {
		t.Fatalf("errors = %v, want one at path \"createPost\"", errs)
	}
}

// ------------------------------------------------------------ hardening

func TestDepthLimitRejectsACheapButUnboundedQuery(t *testing.T) {
	ts, store := batchedServer(t)

	status, body := postQuery(t, ts,
		`{ posts(first: 1) { author { posts(first: 1) { author { posts(first: 1) { title } } } } } }`)
	requireStatus(t, status, http.StatusBadRequest)

	if _, ok := body["data"]; ok {
		t.Error("a rejected query must not carry a data key: nothing executed")
	}
	errs := responseErrors(t, body)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "too deep") {
		t.Fatalf("errors = %v, want one saying the query is too deep", errs)
	}
	if store.TotalCalls() != 0 {
		t.Errorf("store was called %d time(s): the limit must run before any resolver", store.TotalCalls())
	}
}

func TestComplexityLimitRejectsAWideQuery(t *testing.T) {
	ts, store := batchedServer(t)

	status, body := postQuery(t, ts, `{ posts(first: 5000) { title } }`)
	requireStatus(t, status, http.StatusBadRequest)

	errs := responseErrors(t, body)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "too complex") {
		t.Fatalf("errors = %v, want one saying the query is too complex", errs)
	}
	if store.TotalCalls() != 0 {
		t.Errorf("store was called %d time(s): the limit must run before any resolver", store.TotalCalls())
	}
}

func TestRequestErrorsAreRejectedBeforeExecution(t *testing.T) {
	ts, _ := batchedServer(t)
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"malformed", `{ posts(first: 1) { title `, "expected"},
		{"unknown field", `{ posts(first: 1) { colour } }`, "colour"},
		{"scalar with a selection", `{ posts(first: 1) { title { nope } } }`, "scalar"},
		{"object without a selection", `{ posts(first: 1) { author } }`, "subfields"},
		{"variables", `query Q { posts(first: $n) { title } }`, "variables"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body := postQuery(t, ts, tc.query)
			requireStatus(t, status, http.StatusBadRequest)
			if _, ok := body["data"]; ok {
				t.Error("a request error must not carry a data key")
			}
			errs := responseErrors(t, body)
			if len(errs) != 1 || !strings.Contains(errs[0].Message, tc.want) {
				t.Fatalf("errors = %v, want one mentioning %q", errs, tc.want)
			}
		})
	}
}

func TestTransportRules(t *testing.T) {
	ts, _ := batchedServer(t)

	t.Run("GET is not allowed", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/graphql")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", resp.StatusCode)
		}
	})

	t.Run("wrong content type", func(t *testing.T) {
		status, _ := postRaw(t, ts, "text/plain", []byte(`{"query":"{ posts { id } }"}`))
		requireStatus(t, status, http.StatusUnsupportedMediaType)
	})

	t.Run("oversized document", func(t *testing.T) {
		store := NewStore()
		small := httptest.NewServer(NewHandler(Config{
			Schema: NewSchema(store), Store: store,
			Limits: testLimits(), MaxBodyBytes: 128, Logger: discardLogger(),
		}))
		defer small.Close()

		body, err := json.Marshal(map[string]string{
			"query": "{ posts { " + strings.Repeat("id ", 100) + "} }",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		resp, err := small.Client().Post(small.URL+"/graphql", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("status = %d, want 413", resp.StatusCode)
		}
	})
}

// ------------------------------------------------------------ request scope

// A loader's cache is a per-request promise, not a cache. Two requests, two
// batches — the second one must not serve data the first one read.
func TestLoadersDoNotOutliveARequest(t *testing.T) {
	ts, store := batchedServer(t)

	postQuery(t, ts, qPostsWithAuthors)
	postQuery(t, ts, qPostsWithAuthors)

	requireCallCount(t, store, "AuthorsByIDs", 2)
}

func TestConcurrentRequestsAreRaceFree(t *testing.T) {
	ts, _ := batchedServer(t)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				body, err := json.Marshal(map[string]string{"query": qDeepTree})
				if err != nil {
					t.Error(err)
					return
				}
				resp, err := ts.Client().Post(ts.URL+"/graphql", "application/json", bytes.NewReader(body))
				if err != nil {
					t.Error(err)
					return
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("status = %d, want 200", resp.StatusCode)
					return
				}
			}
		}()
	}
	wg.Wait()
}
