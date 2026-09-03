package inspect

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationErrorMessage(t *testing.T) {
	e := &ValidationError{Field: "email", Reason: "missing @"}
	if got, want := e.Error(), "invalid email: missing @"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestGetFound(t *testing.T) {
	got, err := Get("greeting")
	if err != nil {
		t.Fatalf(`Get("greeting") returned error %v, want nil`, err)
	}
	if want := "hello"; got != want {
		t.Errorf(`Get("greeting") = %q, want %q`, got, want)
	}
}

func TestGetMissingKey(t *testing.T) {
	_, err := Get("ghost")
	if err == nil {
		t.Fatal(`Get("ghost") returned nil error, want a wrapped ErrNotFound`)
	}
	if got, want := err.Error(), `get "ghost": not found`; got != want {
		t.Errorf("err.Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false for %q — did you wrap with %%w?", err)
	}
}

func TestGetEmptyKey(t *testing.T) {
	_, err := Get("")
	if err == nil {
		t.Fatal(`Get("") returned nil error, want a wrapped *ValidationError`)
	}
	if got, want := err.Error(), "get: invalid key: must not be empty"; got != want {
		t.Errorf("err.Error() = %q, want %q", got, want)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("errors.As found no *ValidationError in %q — did you wrap with %%w?", err)
	}
	if ve.Field != "key" {
		t.Errorf("ValidationError.Field = %q, want %q", ve.Field, "key")
	}
}

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"bare sentinel", ErrNotFound, true},
		{"wrapped once", fmt.Errorf("outer: %w", ErrNotFound), true},
		{"wrapped twice", fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrNotFound)), true},
		{"same text, different error", errors.New("not found"), false},
		{"unrelated error", errors.New("boom"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsNotFound(c.err); got != c.want {
				t.Errorf("IsNotFound(%v) = %t, want %t", c.err, got, c.want)
			}
		})
	}
}

func TestInvalidField(t *testing.T) {
	deep := fmt.Errorf("handler: %w",
		fmt.Errorf("get: %w", &ValidationError{Field: "id", Reason: "must be numeric"}))
	cases := []struct {
		name      string
		err       error
		wantField string
		wantOK    bool
	}{
		{"nil error", nil, "", false},
		{"bare validation error", &ValidationError{Field: "email", Reason: "missing @"}, "email", true},
		{"wrapped twice", deep, "id", true},
		{"no validation error in chain", fmt.Errorf("get: %w", ErrNotFound), "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			field, ok := InvalidField(c.err)
			if field != c.wantField || ok != c.wantOK {
				t.Errorf("InvalidField(%v) = (%q, %t), want (%q, %t)",
					c.err, field, ok, c.wantField, c.wantOK)
			}
		})
	}
}
