package harden

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decode runs DecodeJSON the way a handler would and hands back the decoded
// value plus whatever went wrong.
func decode(t *testing.T, contentType, body string) (CreateTaskRequest, error) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	var req CreateTaskRequest
	return req, DecodeJSON(httptest.NewRecorder(), r, DefaultMaxBodyBytes, &req)
}

func requestError(t *testing.T, err error) *RequestError {
	t.Helper()
	var re *RequestError
	if !errors.As(err, &re) {
		t.Fatalf("error %v (%T) is not a *RequestError: handlers cannot map it to a status", err, err)
	}
	if re.Message == "" {
		t.Error("the *RequestError has an empty message")
	}
	return re
}

func TestDecodeJSONAcceptsAValidBody(t *testing.T) {
	for _, ct := range []string{"application/json", "application/json; charset=utf-8"} {
		got, err := decode(t, ct, `{"title":"write tests","priority":2,"tags":["work"]}`)
		if err != nil {
			t.Fatalf("Content-Type %q: unexpected error %v", ct, err)
		}
		if got.Title != "write tests" || got.Priority != 2 || len(got.Tags) != 1 {
			t.Errorf("decoded %+v, want title=%q priority=2 tags=[work]", got, "write tests")
		}
	}
}

func TestDecodeJSONRejectsTheWrongContentType(t *testing.T) {
	for _, ct := range []string{"", "text/plain", "application/x-www-form-urlencoded"} {
		_, err := decode(t, ct, `{"title":"t","priority":1}`)
		re := requestError(t, err)
		if re.Status != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q: status = %d, want 415", ct, re.Status)
		}
	}
}

func TestDecodeJSONMapsMalformedBodiesTo400(t *testing.T) {
	tests := map[string]string{
		"empty body":         ``,
		"truncated object":   `{"title":"t"`,
		"not JSON at all":    `title=t&priority=1`,
		"two JSON values":    `{"title":"a","priority":1}{"title":"b","priority":1}`,
		"trailing garbage":   `{"title":"a","priority":1} oops`,
		"top-level not JSON": `[1,2,3]`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := decode(t, "application/json", body)
			re := requestError(t, err)
			if re.Status != http.StatusBadRequest {
				t.Errorf("status = %d, want 400: the body is not one well-formed JSON object", re.Status)
			}
		})
	}
}

func TestDecodeJSONRejectsUnknownFieldsAndNamesThem(t *testing.T) {
	_, err := decode(t, "application/json", `{"title":"t","priority":1,"colour":"red"}`)
	re := requestError(t, err)
	if re.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", re.Status)
	}
	if len(re.Fields) != 1 || re.Fields[0].Field != "colour" {
		t.Errorf("fields = %+v, want one field error naming %q: a client with a typo needs to be told which key", re.Fields, "colour")
	}
}

func TestDecodeJSONRejectsWrongTypesAndNamesTheField(t *testing.T) {
	_, err := decode(t, "application/json", `{"title":"t","priority":"high"}`)
	re := requestError(t, err)
	if re.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: a string where a number belongs is malformed, not merely invalid", re.Status)
	}
	if len(re.Fields) != 1 || re.Fields[0].Field != "priority" {
		t.Errorf("fields = %+v, want one field error naming %q", re.Fields, "priority")
	}
}

func TestDecodeJSONEnforcesTheSizeLimit(t *testing.T) {
	body := `{"title":"` + strings.Repeat("a", 4000) + `","priority":1}`
	r := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var req CreateTaskRequest
	err := DecodeJSON(httptest.NewRecorder(), r, 1024, &req)

	re := requestError(t, err)
	if re.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413: http.MaxBytesReader must cap the body before it is buffered", re.Status)
	}
}

// Well-formed JSON that breaks a business rule is a different failure from
// JSON the parser could not read, and it deserves a different status.
func TestDecodeJSONReportsValidationFailuresAs422(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		fields []string
	}{
		{"blank title", `{"title":"   ","priority":3}`, []string{"title"}},
		{"long title", `{"title":"` + strings.Repeat("x", 81) + `","priority":3}`, []string{"title"}},
		{"priority too low", `{"title":"t","priority":0}`, []string{"priority"}},
		{"priority too high", `{"title":"t","priority":6}`, []string{"priority"}},
		{"too many tags", `{"title":"t","priority":3,"tags":["a","b","c","d","e","f"]}`, []string{"tags"}},
		{"everything wrong", `{"title":"","priority":9,"tags":["a","b","c","d","e","f"]}`,
			[]string{"title", "priority", "tags"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decode(t, "application/json", tc.body)
			re := requestError(t, err)
			if re.Status != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", re.Status)
			}
			if len(re.Fields) != len(tc.fields) {
				t.Fatalf("fields = %+v, want %d (%v): report every failure, not the first", re.Fields, len(tc.fields), tc.fields)
			}
			for i, want := range tc.fields {
				if re.Fields[i].Field != want {
					t.Errorf("fields[%d].Field = %q, want %q (declaration order)", i, re.Fields[i].Field, want)
				}
				if re.Fields[i].Message == "" {
					t.Errorf("fields[%d] has no message: %q alone does not tell a user what to change", i, want)
				}
			}
		})
	}
}

// An 80-rune title of multi-byte characters is 240 bytes but still 80
// characters, and the limit is about characters.
func TestValidateCountsRunesNotBytes(t *testing.T) {
	req := CreateTaskRequest{Title: strings.Repeat("é", 80), Priority: 1}
	if got := req.Validate(); len(got) != 0 {
		t.Errorf("Validate() = %+v, want no errors: 80 accented characters are 80 characters", got)
	}
}

func TestCreateTaskHandlerAnswersTheDecodeError(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"t","priority":99}`))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	CreateTaskHandler(rec, r)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	env := decodeEnvelope(t, rec)
	if len(env.Error.Fields) != 1 || env.Error.Fields[0].Field != "priority" {
		t.Errorf("body fields = %+v, want one entry for %q", env.Error.Fields, "priority")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestCreateTaskHandlerAcceptsAGoodRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"ship it","priority":1}`))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	CreateTaskHandler(rec, r)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}
}
