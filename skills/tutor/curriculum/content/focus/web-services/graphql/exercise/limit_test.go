package gql

import (
	"strings"
	"testing"
)

func TestDepth(t *testing.T) {
	s := NewSchema(NewStore())
	tests := []struct {
		query string
		want  int
	}{
		{`{ posts(first: 1) { title } }`, 2},
		{`{ post(id: "p1") { author { name } } }`, 3},
		{`{ posts(first: 1) { title comments(first: 1) { body author { name } } } }`, 4},
		{`{ posts(first: 1) { author { posts(first: 1) { author { posts(first: 1) { title } } } } } }`, 6},
		{`mutation { createPost(authorId: "a1", title: "x") { id } }`, 2},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			if got := Depth(mustParse(t, s, tc.query)); got != tc.want {
				t.Errorf("Depth = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestComplexity(t *testing.T) {
	s := NewSchema(NewStore())
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"single object", `{ post(id: "p1") { title } }`, 2},
		{"bounded list", `{ posts(first: 3) { title } }`, 4},
		{"unbounded list uses the page size", `{ posts { title } }`, 21},
		{"nested lists multiply", `{ posts(first: 2) { title comments(first: 3) { body } } }`, 11},
		{"the abusive one", `{ posts(first: 100) { comments(first: 100) { author { name } } } }`, 20101},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Complexity(s, mustParse(t, s, tc.query)); got != tc.want {
				t.Errorf("Complexity = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestComplexityHonoursFieldWeight(t *testing.T) {
	s := NewSchema(NewStore())
	op := mustParse(t, s, `{ posts(first: 3) { title } }`)

	s.Types["Post"].Fields["title"].Weight = 4
	if got, want := Complexity(s, op), 1+3*4; got != want {
		t.Errorf("Complexity with a weighted field = %d, want %d: a field you know is expensive should cost more", got, want)
	}
}

func TestLimitsCheck(t *testing.T) {
	s := NewSchema(NewStore())
	tests := []struct {
		name    string
		query   string
		limits  Limits
		wantErr string
	}{
		{
			name:   "a modest query passes",
			query:  qPostsWithAuthors,
			limits: testLimits(),
		},
		{
			name:    "cheap but deep is still refused",
			query:   `{ posts(first: 1) { author { posts(first: 1) { author { posts(first: 1) { title } } } } } }`,
			limits:  testLimits(),
			wantErr: "too deep",
		},
		{
			name:    "shallow but expensive is refused too",
			query:   `{ posts(first: 5000) { title } }`,
			limits:  testLimits(),
			wantErr: "too complex",
		},
		{
			name:   "a non-positive limit disables that check",
			query:  `{ posts(first: 5000) { title } }`,
			limits: Limits{MaxDepth: 5},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.limits.Check(s, mustParse(t, s, tc.query))
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("Check = %v, want nil", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("Check = nil, want an error mentioning %q", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("Check = %q, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}
