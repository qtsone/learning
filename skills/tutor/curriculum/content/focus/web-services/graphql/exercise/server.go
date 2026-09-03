package gql

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
)

// DefaultMaxBodyBytes bounds the query document. GraphQL queries are text a
// stranger writes, so the byte bound from the hardening lesson applies here
// exactly as it did there — and here it is also the first bound on how much
// work the parser and the complexity analyser can be made to do.
const DefaultMaxBodyBytes int64 = 64 << 10

// Config is everything the endpoint needs.
type Config struct {
	Schema       *Schema
	Store        *Store
	Limits       Limits
	MaxBodyBytes int64
	Logger       *slog.Logger
}

// request is the GraphQL-over-HTTP request body. OperationName is accepted and
// ignored (this executor takes one operation per document); Variables are
// refused rather than silently dropped, because a client whose variables
// vanish gets wrong answers instead of an error.
type request struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

// NewHandler returns the whole API: one endpoint, one method.
//
// One endpoint is the defining operational fact about GraphQL. Everything a
// client can ask goes through POST /graphql, which means no URL identifies a
// resource, which means an HTTP cache — browser, CDN, reverse proxy — has
// nothing to key on and nothing it is allowed to store. The caching you get
// for free in REST becomes something you build.
func NewHandler(cfg Config) http.Handler {
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = DefaultMaxBodyBytes
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	mux := http.NewServeMux()
	// The 1.22 method pattern answers anything but POST with 405 and an Allow
	// header. The spec does permit GET for queries — which would let caches
	// work — at the price of every query being URL-length-bounded and every
	// mutation needing a second path. Pick one deliberately; this endpoint is
	// POST only.
	mux.Handle("POST /graphql", graphqlHandler(cfg))
	return mux
}

func graphqlHandler(cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "" {
			mt, _, err := mime.ParseMediaType(ct)
			if err != nil || mt != "application/json" {
				requestError(w, cfg.Logger, http.StatusUnsupportedMediaType,
					"content-type must be application/json")
				return
			}
		}

		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBodyBytes)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var req request
		if err := dec.Decode(&req); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				requestError(w, cfg.Logger, http.StatusRequestEntityTooLarge, "query document too large")
				return
			}
			requestError(w, cfg.Logger, http.StatusBadRequest, "request body must be a JSON object with a query field")
			return
		}
		if len(req.Variables) > 0 {
			requestError(w, cfg.Logger, http.StatusBadRequest, "variables are not supported by this executor")
			return
		}

		op, err := Parse(req.Query)
		if err != nil {
			requestError(w, cfg.Logger, http.StatusBadRequest, err.Error())
			return
		}
		if err := Validate(cfg.Schema, op); err != nil {
			requestError(w, cfg.Logger, http.StatusBadRequest, err.Error())
			return
		}
		// The security gate. It runs after validation (which is linear in the
		// size of a body you have already bounded) and before a single
		// resolver, because after the first resolver the work has begun.
		if err := cfg.Limits.Check(cfg.Schema, op); err != nil {
			cfg.Logger.Warn("graphql query rejected",
				"reason", err.Error(),
				"depth", Depth(op),
				"complexity", Complexity(cfg.Schema, op))
			writeJSON(w, http.StatusBadRequest, &Response{Errors: []QueryError{{Message: err.Error()}}})
			return
		}

		// Loaders live for exactly one request.
		ctx := WithLoaders(r.Context(), NewLoaders(cfg.Store))
		resp := Execute(ctx, cfg.Schema, op)

		// Execution ran, so this is a 200 whatever is in the errors array. A
		// GraphQL response is not a status code: half the tree may have
		// resolved, and the client is expected to read both keys.
		writeJSON(w, http.StatusOK, resp)
	})
}

func requestError(w http.ResponseWriter, log *slog.Logger, status int, msg string) {
	log.Warn("graphql request rejected", "status", status, "reason", msg)
	writeJSON(w, status, &Response{Errors: []QueryError{{Message: msg}}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
