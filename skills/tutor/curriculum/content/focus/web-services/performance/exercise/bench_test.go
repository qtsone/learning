package apiperf

import (
	"context"
	"strings"
	"testing"
	"time"
)

// These benchmarks are not part of the grade — `go test` does not run them and
// nothing here asserts a duration. They are here for you to run by hand, read,
// and explain:
//
//	go test -run '^$' -bench . -benchmem ./...
//
// Two rules from the lesson apply to every number they print. Compare like
// with like (same machine, same dataset, nothing else running), and treat a
// single run as noise: `-count=10` and `benchstat` exist because one number is
// not a measurement.
//
// The comparisons worth having an opinion about before you look:
//
//	OffsetDeep vs KeysetDeep     — what does the database do that the wall
//	                               clock is showing you?
//	AuthorsOneByOne vs Batched   — the N+1, in nanoseconds instead of counters.
//	FeedUncached vs FeedCached   — what is left once the database is gone?

func benchStore(b *testing.B, authors, articles int) (*Store, *fakeClock) {
	b.Helper()
	st, clk := newTestStore(b)
	as := make([]Author, 0, authors)
	for i := range authors {
		a, err := st.CreateAuthor(context.Background(), "author-"+strings.Repeat("x", i%5))
		if err != nil {
			b.Fatalf("CreateAuthor: %v", err)
		}
		as = append(as, a)
	}
	for i := range articles {
		if _, err := st.CreateArticle(context.Background(), as[i%len(as)].ID,
			"article", strings.Repeat("body ", 40)); err != nil {
			b.Fatalf("CreateArticle: %v", err)
		}
		clk.Advance(time.Second)
	}
	return st, clk
}

const benchDepth = 5000

func BenchmarkListArticlesOffsetDeep(b *testing.B) {
	st, _ := benchStore(b, 20, benchDepth+100)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		if _, err := st.ListArticlesOffset(ctx, 20, benchDepth); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListArticlesKeysetDeep(b *testing.B) {
	st, _ := benchStore(b, 20, benchDepth+100)
	ctx := context.Background()
	deep, err := st.ListArticlesOffset(ctx, 1, benchDepth)
	if err != nil || len(deep) == 0 {
		b.Skip("keyset benchmark needs the seeded rows; implement ListArticlesOffset's data first")
	}
	cursor := &Cursor{CreatedAt: deep[0].CreatedAt, ID: deep[0].ID}
	b.ResetTimer()
	for range b.N {
		if _, err := st.ListArticles(ctx, 20, cursor); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuthorsOneByOne(b *testing.B) {
	st, _ := benchStore(b, 20, 100)
	ctx := context.Background()
	ids := make([]int64, 0, 20)
	for i := int64(1); i <= 20; i++ {
		ids = append(ids, i)
	}
	b.ResetTimer()
	for range b.N {
		for _, id := range ids {
			if _, err := st.AuthorByID(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkAuthorsBatched(b *testing.B) {
	st, _ := benchStore(b, 20, 100)
	ctx := context.Background()
	ids := make([]int64, 0, 20)
	for i := int64(1); i <= 20; i++ {
		ids = append(ids, i)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := st.AuthorsByIDs(ctx, ids); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFeedUncached(b *testing.B) {
	st, clk := benchStore(b, 20, 2000)
	svc := NewService(st, 8, 30*time.Second, clk)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		svc.Cache.Purge()
		if _, err := svc.FeedJSON(ctx, 20, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFeedCached(b *testing.B) {
	st, clk := benchStore(b, 20, 2000)
	svc := NewService(st, 8, 30*time.Second, clk)
	ctx := context.Background()
	if _, err := svc.FeedJSON(ctx, 20, ""); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := svc.FeedJSON(ctx, 20, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkETag(b *testing.B) {
	body := []byte(strings.Repeat("{\"id\":1,\"title\":\"a fairly typical feed item\"},", 1200))
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for range b.N {
		ETag(body)
	}
}
