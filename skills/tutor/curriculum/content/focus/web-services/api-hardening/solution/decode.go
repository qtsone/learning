package harden

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// DefaultMaxBodyBytes is the ceiling for a JSON request body on this API.
// 64 KiB is generous for a form submission and tiny compared with the memory
// an unbounded decode can consume.
const DefaultMaxBodyBytes int64 = 64 << 10

// unknownFieldPrefix is how encoding/json reports a rejected field. There is
// no typed error for it, so the string is load-bearing; keeping the ugliness
// in one constant means one place to fix if it ever changes.
const unknownFieldPrefix = `json: unknown field `

// Validator is implemented by request types that can check their own fields.
// Returning the failures instead of the first one lets a client fix a whole
// form in one round trip.
type Validator interface {
	Validate() []FieldError
}

// DecodeJSON reads exactly one JSON value from r's body into dst and, when dst
// implements Validator, checks it. Every failure it returns is a
// *RequestError carrying the status the caller should receive.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) error {
	// Requiring the media type is cheap and it closes a door: a body sent as
	// text/plain is a body a browser can post from a form without a preflight.
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &RequestError{
			Status:  http.StatusUnsupportedMediaType,
			Message: "Content-Type must be application/json",
		}
	}

	// MaxBytesReader, not io.LimitReader: LimitReader reports a clean EOF, so
	// a truncated body looks like malformed JSON. MaxBytesReader returns a
	// distinguishable error and tells the server to stop reading.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	// A second value in the body means the client sent something we did not
	// agree to parse — often a sign of a smuggled or double-encoded payload.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &RequestError{
			Status:  http.StatusBadRequest,
			Message: "request body must contain a single JSON object",
		}
	}

	// Parsing answers "could I read it"; validation answers "does it mean
	// anything". Two questions, two statuses.
	if v, ok := dst.(Validator); ok {
		if fields := v.Validate(); len(fields) > 0 {
			return &RequestError{
				Status:  http.StatusUnprocessableEntity,
				Message: "validation failed",
				Fields:  fields,
			}
		}
	}
	return nil
}

// decodeError turns an encoding/json failure into a status and, where the
// error knows which key broke, a field-level message.
func decodeError(err error) *RequestError {
	var (
		maxBytesErr *http.MaxBytesError
		syntaxErr   *json.SyntaxError
		typeErr     *json.UnmarshalTypeError
	)
	switch {
	case errors.As(err, &maxBytesErr):
		return &RequestError{
			Status:  http.StatusRequestEntityTooLarge,
			Message: fmt.Sprintf("request body must not exceed %d bytes", maxBytesErr.Limit),
		}

	case errors.As(err, &syntaxErr):
		return &RequestError{
			Status:  http.StatusBadRequest,
			Message: fmt.Sprintf("malformed JSON at byte %d", syntaxErr.Offset),
		}

	case errors.Is(err, io.ErrUnexpectedEOF):
		return &RequestError{Status: http.StatusBadRequest, Message: "malformed JSON: body ended early"}

	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return &RequestError{Status: http.StatusBadRequest, Message: "request body must be a JSON object"}
		}
		return &RequestError{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
			Fields: []FieldError{{
				Field:   typeErr.Field,
				Message: fmt.Sprintf("must be a %s", typeErr.Type),
			}},
		}

	case errors.Is(err, io.EOF):
		return &RequestError{Status: http.StatusBadRequest, Message: "request body must not be empty"}

	case strings.HasPrefix(err.Error(), unknownFieldPrefix):
		// encoding/json quotes the name, so unquoting it is the reverse of
		// how the message was built.
		name, uerr := strconv.Unquote(strings.TrimPrefix(err.Error(), unknownFieldPrefix))
		if uerr != nil {
			return &RequestError{Status: http.StatusBadRequest, Message: "invalid request body"}
		}
		return &RequestError{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
			Fields:  []FieldError{{Field: name, Message: "unknown field"}},
		}

	default:
		// Never hand the raw decoder text to the caller: it can echo the body
		// back, and the body may be someone's data.
		return &RequestError{Status: http.StatusBadRequest, Message: "invalid request body"}
	}
}

// CreateTaskRequest is the body of POST /tasks.
type CreateTaskRequest struct {
	Title    string   `json:"title"`
	Priority int      `json:"priority"`
	Tags     []string `json:"tags"`
}

const (
	maxTitleRunes = 80
	maxTags       = 5
)

// Validate reports every field that breaks a rule, in declaration order, so
// the response is stable enough to test and to show a user.
func (req CreateTaskRequest) Validate() []FieldError {
	var errs []FieldError

	switch {
	case strings.TrimSpace(req.Title) == "":
		errs = append(errs, FieldError{Field: "title", Message: "must not be empty"})
	case utf8.RuneCountInString(req.Title) > maxTitleRunes:
		// Runes, not bytes: "é" is two bytes and one character, and the
		// limit is a promise about what the user typed.
		errs = append(errs, FieldError{
			Field:   "title",
			Message: fmt.Sprintf("must be at most %d characters", maxTitleRunes),
		})
	}

	if req.Priority < 1 || req.Priority > 5 {
		errs = append(errs, FieldError{Field: "priority", Message: "must be between 1 and 5"})
	}

	if len(req.Tags) > maxTags {
		errs = append(errs, FieldError{
			Field:   "tags",
			Message: fmt.Sprintf("must contain at most %d tags", maxTags),
		})
	}

	return errs
}
