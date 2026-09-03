package gql

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrNotFound is the store's answer for an id nobody has.
var ErrNotFound = errors.New("not found")

type Author struct {
	ID   string
	Name string
}

type Post struct {
	ID       string
	AuthorID string
	Title    string
}

type Comment struct {
	ID       string
	PostID   string
	AuthorID string
	Body     string
}

// Call is one trip to the store. The tests count these: an N+1 is not a
// feeling, it is a number, and this is where the number comes from.
type Call struct {
	Method string
	Keys   []string
}

// Store stands in for the database. It is deliberately in-memory and
// deliberately instrumented; everything else about it behaves like a real one,
// including being shared by every request goroutine, which is why every field
// is guarded by mu.
type Store struct {
	mu       sync.Mutex
	authors  map[string]Author
	posts    []Post
	comments []Comment
	nextPost int
	calls    []Call
}

// NewStore returns the fixture data every test runs against: two real authors,
// five posts (the last one written by an author id that does not exist), and
// four comments spread unevenly across the posts.
func NewStore() *Store {
	return &Store{
		authors: map[string]Author{
			"a1": {ID: "a1", Name: "Ada Lovelace"},
			"a2": {ID: "a2", Name: "Grace Hopper"},
		},
		posts: []Post{
			{ID: "p1", AuthorID: "a1", Title: "Notes on the Analytical Engine"},
			{ID: "p2", AuthorID: "a2", Title: "Compilers, and why"},
			{ID: "p3", AuthorID: "a1", Title: "On loops"},
			{ID: "p4", AuthorID: "a2", Title: "Debugging by bisection"},
			{ID: "p5", AuthorID: "ghost", Title: "The orphaned draft"},
		},
		comments: []Comment{
			{ID: "c1", PostID: "p1", AuthorID: "a2", Body: "This clarified the loop for me"},
			{ID: "c2", PostID: "p1", AuthorID: "a1", Body: "Corrected the second table"},
			{ID: "c3", PostID: "p2", AuthorID: "a1", Body: "Which pass does the folding?"},
			{ID: "c4", PostID: "p4", AuthorID: "a2", Body: "Halving the input works every time"},
		},
		nextPost: 6,
	}
}

// record notes one call and honours cancellation: a client that hung up should
// not still be making the database work. Every store method starts here.
func (s *Store) record(ctx context.Context, method string, keys ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, Call{Method: method, Keys: append([]string(nil), keys...)})
	return nil
}

func (s *Store) Posts(ctx context.Context) ([]Post, error) {
	if err := s.record(ctx, "Posts"); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Post(nil), s.posts...), nil
}

func (s *Store) PostByID(ctx context.Context, id string) (Post, error) {
	if err := s.record(ctx, "PostByID", id); err != nil {
		return Post{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.posts {
		if p.ID == id {
			return p, nil
		}
	}
	return Post{}, fmt.Errorf("post %q: %w", id, ErrNotFound)
}

// AuthorByID is the singular fetch. One call, one row — and one call per
// parent when a resolver reaches for it inside a list.
func (s *Store) AuthorByID(ctx context.Context, id string) (Author, error) {
	if err := s.record(ctx, "AuthorByID", id); err != nil {
		return Author{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.authors[id]
	if !ok {
		return Author{}, fmt.Errorf("author %q: %w", id, ErrNotFound)
	}
	return a, nil
}

// AuthorsByIDs is the plural fetch a dataloader batches into. It returns a map
// rather than a slice on purpose: a slice makes the caller responsible for
// keeping results in key order, and that off-by-one is the classic dataloader
// bug. Keys with no row are simply absent, and the loader turns that absence
// into a per-key ErrNotFound.
func (s *Store) AuthorsByIDs(ctx context.Context, ids []string) (map[string]Author, error) {
	if err := s.record(ctx, "AuthorsByIDs", ids...); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Author, len(ids))
	for _, id := range ids {
		if a, ok := s.authors[id]; ok {
			out[id] = a
		}
	}
	return out, nil
}

func (s *Store) CommentsByPost(ctx context.Context, postID string) ([]Comment, error) {
	if err := s.record(ctx, "CommentsByPost", postID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Comment
	for _, c := range s.comments {
		if c.PostID == postID {
			out = append(out, c)
		}
	}
	return out, nil
}

// CommentsByPostIDs is a one-to-many batch. Note that every requested id gets
// an entry, empty slice included: "this post has no comments" and "there is no
// such post" are different answers, and omitting the key would merge them.
func (s *Store) CommentsByPostIDs(ctx context.Context, postIDs []string) (map[string][]Comment, error) {
	if err := s.record(ctx, "CommentsByPostIDs", postIDs...); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]Comment, len(postIDs))
	for _, id := range postIDs {
		out[id] = []Comment{}
	}
	for _, c := range s.comments {
		if _, ok := out[c.PostID]; ok {
			out[c.PostID] = append(out[c.PostID], c)
		}
	}
	return out, nil
}

func (s *Store) PostsByAuthor(ctx context.Context, authorID string) ([]Post, error) {
	if err := s.record(ctx, "PostsByAuthor", authorID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Post
	for _, p := range s.posts {
		if p.AuthorID == authorID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) PostsByAuthorIDs(ctx context.Context, authorIDs []string) (map[string][]Post, error) {
	if err := s.record(ctx, "PostsByAuthorIDs", authorIDs...); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]Post, len(authorIDs))
	for _, id := range authorIDs {
		out[id] = []Post{}
	}
	for _, p := range s.posts {
		if _, ok := out[p.AuthorID]; ok {
			out[p.AuthorID] = append(out[p.AuthorID], p)
		}
	}
	return out, nil
}

func (s *Store) CreatePost(ctx context.Context, authorID, title string) (Post, error) {
	if err := s.record(ctx, "CreatePost", authorID); err != nil {
		return Post{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.authors[authorID]; !ok {
		return Post{}, fmt.Errorf("author %q: %w", authorID, ErrNotFound)
	}
	p := Post{ID: fmt.Sprintf("p%d", s.nextPost), AuthorID: authorID, Title: title}
	s.nextPost++
	s.posts = append(s.posts, p)
	return p, nil
}

// Calls returns every recorded call to method, in order.
func (s *Store) Calls(method string) []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Call
	for _, c := range s.calls {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

// CallCount reports how many times method was called.
func (s *Store) CallCount(method string) int { return len(s.Calls(method)) }

// TotalCalls reports every call to the store, whatever the method.
func (s *Store) TotalCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// ResetCalls clears the recorded calls without touching the data.
func (s *Store) ResetCalls() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = nil
}
