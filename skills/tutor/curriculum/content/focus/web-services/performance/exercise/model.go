package apiperf

import "time"

// Author writes articles.
type Author struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Article is a row of the articles table.
type Article struct {
	ID        int64     `json:"id"`
	AuthorID  int64     `json:"author_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// FeedItem is what GET /articles returns per row: the article joined with its
// author, and deliberately without the body. A feed that ships every body is
// the cheapest performance bug there is — you pay for it in bytes, in
// compression CPU and in the client's parse time, for data nobody rendered.
type FeedItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// Page is one page of the feed. NextCursor is empty when there is no next
// page, which is how a client knows to stop without making the empty request.
type Page struct {
	Items      []FeedItem `json:"items"`
	NextCursor string     `json:"next_cursor"`
}
