package apiperf

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

// Service is the feed API: a store, a cache in front of it, and how long a
// cached page may be served for.
type Service struct {
	Store  *Store
	Cache  *Cache
	MaxAge time.Duration
}

// NewService wires a service with a cache of maxEntries pages, each valid for
// maxAge, expiring against clock.
func NewService(store *Store, maxEntries int, maxAge time.Duration, clock Clock) *Service {
	return &Service{
		Store:  store,
		Cache:  NewCache(maxEntries, maxAge, clock),
		MaxAge: maxAge,
	}
}

// cacheKey identifies a page by everything that changes its bytes. Miss one
// input and you serve the wrong page to somebody; add an input nobody varies
// and your hit rate collapses. If this endpoint ever became per-user, the user
// id would belong here — and then you would ask whether a shared cache is
// still the right shape at all.
func cacheKey(limit int, cursor string) string {
	return strconv.Itoa(limit) + "|" + cursor
}

// Feed returns one page of the feed: at most limit items, newest first,
// starting after cursor (empty cursor means the first page).
//
// The contract the tests enforce:
//
//   - it makes exactly **two** database round trips for a non-empty page,
//     whatever limit is — one for the articles, one for their authors — and
//     one round trip when the page comes back empty;
//   - NextCursor is set only when there is genuinely a next page. Ask the
//     store for limit+1 rows, and if you get them, drop the extra and encode a
//     cursor from the last row you keep. Setting a cursor from the last row of
//     every page costs the client one empty request at the end of the feed;
//   - Items is never nil, so an empty page marshals as [] rather than null.
//     Client code that does `for (const x of body.items)` deserves better than
//     a null;
//   - an unreadable cursor comes back as an error matching ErrBadCursor.
//
// What is here now is the version almost everybody writes first: it is
// correct for the first page, it pages with OFFSET, and it asks the database
// for one author per article. Both problems are measured by the tests.
func (s *Service) Feed(ctx context.Context, limit int, cursor string) (Page, error) {
	// TODO: rewrite. Decode the cursor, call ListArticles for limit+1 rows,
	// collect the author ids, fetch them with AuthorsByIDs, and assemble.
	articles, err := s.Store.ListArticlesOffset(ctx, limit, 0)
	if err != nil {
		return Page{}, err
	}
	items := make([]FeedItem, 0, len(articles))
	for _, a := range articles {
		author, err := s.Store.AuthorByID(ctx, a.AuthorID)
		if err != nil {
			return Page{}, err
		}
		items = append(items, FeedItem{
			ID:        a.ID,
			Title:     a.Title,
			Author:    author,
			CreatedAt: a.CreatedAt,
		})
	}
	return Page{Items: items}, nil
}

// FeedJSON returns the rendered page and its ETag, served from the cache when
// a fresh copy is there.
//
// A hit must do no database work at all — that is the whole point, and a test
// asserts it by counting queries. A miss renders the page once and stores the
// bytes and the tag together, because they must never disagree.
func (s *Service) FeedJSON(ctx context.Context, limit int, cursor string) (CachedPage, error) {
	// TODO: check the cache first, and store the result on a miss.
	page, err := s.Feed(ctx, limit, cursor)
	if err != nil {
		return CachedPage{}, err
	}
	body, err := json.Marshal(page)
	if err != nil {
		return CachedPage{}, err
	}
	return CachedPage{Body: body, ETag: ETag(body)}, nil
}
