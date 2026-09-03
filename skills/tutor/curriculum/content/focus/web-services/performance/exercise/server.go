package apiperf

import (
	"net/http"
	"strconv"
)

// Routes is the whole HTTP surface, using the 1.22 method+pattern mux from S5.
func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles", s.handleListArticles)
	mux.HandleFunc("POST /articles", s.handleCreateArticle)
	return mux
}

// cacheControl is the freshness the API promises to clients and to any cache
// between you and them. It is deliberately the same duration as the in-process
// cache's TTL: two caches with different ideas of "stale" is a bug report you
// cannot reproduce.
//
// "public" is a claim that this response is the same for everybody. The moment
// this feed became per-user — anything behind the session cookie from the
// authentication lesson — it would have to become "private", or a shared cache
// would be entitled to hand one user's page to the next.
func (s *Service) cacheControl() string {
	return "public, max-age=" + strconv.Itoa(int(s.MaxAge.Seconds()))
}

// createArticleRequest is the body of POST /articles.
type createArticleRequest struct {
	AuthorID int64  `json:"author_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

// handleListArticles serves GET /articles?limit=&cursor=.
//
// The order of operations is the lesson:
//
//  1. parse limit (parseLimit) and read the cursor parameter;
//  2. get the rendered page from FeedJSON;
//  3. map an ErrBadCursor to 400 and anything else to the 500 writeError
//     already gives you;
//  4. set ETag and Cache-Control — on **every** answer, including the 304,
//     because a 304 that drops them tells the client its cached copy has no
//     tag and no lifetime;
//  5. if MatchETag says the client already has these bytes, answer 304 and
//     write no body;
//  6. otherwise set Content-Type and write the bytes.
//
// A 304 that arrives after a cache hit is the cheapest thing this service
// does: no query, no marshal, no body.
func (s *Service) handleListArticles(w http.ResponseWriter, r *http.Request) {
	// TODO: implement the six steps above.
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := s.FeedJSON(r.Context(), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(page.Body)
}

// handleCreateArticle serves POST /articles.
//
// It must: decode the body with decodeJSON; reject a blank title with 400;
// reject an author_id with no row with 400 (ErrNotFound from AuthorByID is a
// bad request, not a server error); insert; **invalidate the cache**; and
// answer 201 with the created Article.
//
// Step five is the one that gets forgotten, and it is why a stale feed is a
// correctness bug rather than a performance detail: without it, a client that
// just created an article does not see it for MaxAge seconds and reports your
// API as broken.
func (s *Service) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	// TODO: implement.
	writeError(w, &requestError{Status: http.StatusNotImplemented, Message: "not implemented"})
}
