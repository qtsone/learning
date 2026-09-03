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

// cacheKey identifies a page by everything that changes its bytes.
func cacheKey(limit int, cursor string) string {
	return strconv.Itoa(limit) + "|" + cursor
}

// Feed returns one page of the feed: at most limit items, newest first,
// starting after cursor (empty cursor means the first page).
//
// Two round trips, whatever limit is. The limit+1 read is what lets NextCursor
// mean "there is definitely more" instead of "there might be": without it,
// every client walks off the end of the feed with one extra empty request, and
// on a feed that is polled that is a measurable share of your traffic.
func (s *Service) Feed(ctx context.Context, limit int, cursor string) (Page, error) {
	var after *Cursor
	if cursor != "" {
		decoded, err := DecodeCursor(cursor)
		if err != nil {
			return Page{}, err
		}
		after = &decoded
	}

	articles, err := s.Store.ListArticles(ctx, limit+1, after)
	if err != nil {
		return Page{}, err
	}

	next := ""
	if len(articles) > limit {
		articles = articles[:limit]
		last := articles[len(articles)-1]
		next = EncodeCursor(Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}

	authorIDs := make([]int64, 0, len(articles))
	for _, a := range articles {
		authorIDs = append(authorIDs, a.AuthorID)
	}
	authors, err := s.Store.AuthorsByIDs(ctx, authorIDs)
	if err != nil {
		return Page{}, err
	}

	items := make([]FeedItem, 0, len(articles))
	for _, a := range articles {
		items = append(items, FeedItem{
			ID:        a.ID,
			Title:     a.Title,
			Author:    authors[a.AuthorID],
			CreatedAt: a.CreatedAt,
		})
	}
	return Page{Items: items, NextCursor: next}, nil
}

// FeedJSON returns the rendered page and its ETag, served from the cache when
// a fresh copy is there.
//
// The bytes are cached, not the Page: a hit costs a map lookup and a write,
// where caching the struct would still pay for json.Marshal and the hash on
// every request.
//
// Note what this does *not* do: two requests missing on the same key at the
// same instant both render the page. On a hot endpoint that is a stampede, and
// the fix is single-flight — one goroutine renders while the others wait on
// it. It is left out here because it needs a lesson of its own, and because
// measuring your miss rate comes first.
func (s *Service) FeedJSON(ctx context.Context, limit int, cursor string) (CachedPage, error) {
	key := cacheKey(limit, cursor)
	if cached, ok := s.Cache.Get(key); ok {
		return cached, nil
	}

	page, err := s.Feed(ctx, limit, cursor)
	if err != nil {
		return CachedPage{}, err
	}
	body, err := json.Marshal(page)
	if err != nil {
		return CachedPage{}, err
	}

	rendered := CachedPage{Body: body, ETag: ETag(body)}
	s.Cache.Set(key, rendered)
	return rendered, nil
}
