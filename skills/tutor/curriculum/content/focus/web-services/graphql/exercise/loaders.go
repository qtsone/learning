package gql

import (
	"context"
	"errors"
)

// Loaders is the set of dataloaders one request gets. Note "one request": a
// loader's cache is a promise that two reads of the same key inside a single
// query see the same value, and nothing more. Keeping a loader alive across
// requests turns that promise into a stale cache, and a stale cache in an
// authorization-sensitive API hands one user another user's row.
type Loaders struct {
	Authors  *Loader[string, Author]
	Comments *Loader[string, []Comment]
	Posts    *Loader[string, []Post]
}

// NewLoaders wires each loader to the store's plural method. The batch
// function is where "one query per parent" becomes "one query per level".
func NewLoaders(s *Store) *Loaders {
	return &Loaders{
		Authors:  NewLoader(s.AuthorsByIDs),
		Comments: NewLoader(s.CommentsByPostIDs),
		Posts:    NewLoader(s.PostsByAuthorIDs),
	}
}

// DispatchAll runs every loader's pending batch. The executor calls it once
// per level.
func (l *Loaders) DispatchAll(ctx context.Context) {
	l.Authors.Dispatch(ctx)
	l.Comments.Dispatch(ctx)
	l.Posts.Dispatch(ctx)
}

type loadersKey struct{}

// WithLoaders installs a request's loaders. The handler does this once per
// request, immediately after reading the body.
func WithLoaders(ctx context.Context, l *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey{}, l)
}

// LoadersFrom returns the loaders installed for this request, or nil.
func LoadersFrom(ctx context.Context) *Loaders {
	l, _ := ctx.Value(loadersKey{}).(*Loaders)
	return l
}

// loaders is what resolvers call. A missing bundle is a wiring bug rather than
// a client mistake, so it surfaces as a field error instead of a panic in a
// request goroutine.
func loaders(ctx context.Context) (*Loaders, error) {
	l := LoadersFrom(ctx)
	if l == nil {
		return nil, errors.New("no dataloaders in this request context")
	}
	return l, nil
}

// Dispatch is the executor's level boundary: force every loader that has keys
// waiting. It is safe on a context with no loaders (the naive schema needs
// none).
func Dispatch(ctx context.Context) {
	if l := LoadersFrom(ctx); l != nil {
		l.DispatchAll(ctx)
	}
}
