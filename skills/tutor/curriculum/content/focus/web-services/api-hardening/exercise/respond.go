package harden

import (
	"encoding/json"
	"errors"
	"net/http"
)

// FieldError names one field the client got wrong and says what is wrong with
// it. The field name is part of your API contract: clients render these next
// to their form inputs, so renaming one is a breaking change.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RequestError is a fault the caller can fix. It carries the status to send
// and, when the body parsed but broke a rule, the offending fields.
type RequestError struct {
	Status  int
	Message string
	Fields  []FieldError
}

func (e *RequestError) Error() string { return e.Message }

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

// WriteError writes the single error shape this API uses. One shape for every
// failure means a client writes one error path instead of six.
func WriteError(w http.ResponseWriter, status int, message string, fields ...FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{Message: message, Fields: fields}})
}

// WriteRequestError answers a *RequestError with the status it carries.
// Anything else is our bug rather than the caller's, so it becomes a bare 500:
// internal error text leaks table names, file paths and library versions.
func WriteRequestError(w http.ResponseWriter, err error) {
	var re *RequestError
	if errors.As(err, &re) {
		WriteError(w, re.Status, re.Message, re.Fields...)
		return
	}
	WriteError(w, http.StatusInternalServerError, "internal error")
}
