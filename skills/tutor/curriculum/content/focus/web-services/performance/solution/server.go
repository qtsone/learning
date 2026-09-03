package apiperf

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// Routes is the whole HTTP surface, using the 1.22 method+pattern mux from S5.
func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles", s.handleListArticles)
	mux.HandleFunc("POST /articles", s.handleCreateArticle)
	return mux
}

// cacheControl is the freshness the API promises to clients and to any cache
// between you and them, deliberately the same duration as the in-process
// cache's TTL.
//
// "public" is a claim that this response is the same for everybody. A feed
// behind the session cookie from the authentication lesson would have to say
// "private", or a shared cache would be entitled to hand one user's page to
// the next.
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
func (s *Service) handleListArticles(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, err)
		return
	}

	page, err := s.FeedJSON(r.Context(), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		if errors.Is(err, ErrBadCursor) {
			writeError(w, badRequest("invalid cursor"))
			return
		}
		writeError(w, err)
		return
	}

	// Validators go on every answer, the 304 included: a 304 without them
	// tells the client its copy has no tag and no lifetime, and the next
	// request comes back unconditional.
	w.Header().Set("ETag", page.ETag)
	w.Header().Set("Cache-Control", s.cacheControl())
	if MatchETag(r.Header.Get("If-None-Match"), page.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(page.Body)
}

// handleCreateArticle serves POST /articles.
func (s *Service) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	var req createArticleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, badRequest("title must not be blank"))
		return
	}
	if _, err := s.Store.AuthorByID(r.Context(), req.AuthorID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, badRequest("unknown author_id"))
			return
		}
		writeError(w, err)
		return
	}

	article, err := s.Store.CreateArticle(r.Context(), req.AuthorID, req.Title, req.Body)
	if err != nil {
		writeError(w, err)
		return
	}

	// Invalidation. Without this line a client does not see its own write for
	// MaxAge seconds, which is a correctness bug wearing a performance
	// feature's clothes.
	//
	// Purging everything is the honest default: it is one line, and it is
	// never wrong. It is also blunt — with keyset pagination an insert at the
	// head only changes the pages that contain the head, so purging just the
	// entries whose cursor is empty would keep the deep pages warm. Do that
	// when a measurement says the churn matters, not before.
	s.Cache.Purge()

	writeJSON(w, http.StatusCreated, article)
}
