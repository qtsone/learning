package harden

import (
	"net/http"
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
//
// It must, in order:
//
//  1. require a Content-Type of application/json (parameters such as
//     "; charset=utf-8" are fine) and answer 415 otherwise;
//  2. cap the body with http.MaxBytesReader at maxBytes, answering 413;
//  3. decode with DisallowUnknownFields, mapping decoder failures to 400 with
//     a FieldError naming the offending field where the error knows it;
//  4. reject a body holding more than one JSON value with 400;
//  5. run Validate and answer 422 with every failing field.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) error {
	// TODO: implement the five steps above. Map decoder errors with
	// errors.As / errors.Is, not by comparing strings — except for the
	// unknown-field case, which encoding/json only reports as text.
	//
	// The types you need: *http.MaxBytesError, *json.SyntaxError,
	// *json.UnmarshalTypeError, io.ErrUnexpectedEOF, io.EOF.
	return nil
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
	// TODO: title must not be blank and must be at most maxTitleRunes
	// characters (count runes, not bytes); priority must be between 1 and 5;
	// tags must hold at most maxTags entries.
	return nil
}
