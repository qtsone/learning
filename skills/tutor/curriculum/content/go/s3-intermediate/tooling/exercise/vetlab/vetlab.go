// Package vetlab compiles cleanly. That is the point: every bug in here is
// invisible to the compiler. Do not fix anything until go vet has told you
// what it found and you have recorded it in NOTES.md.
package vetlab

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json: "name"`
	Email string `json:"email"`
}

func Describe(u User) string {
	return fmt.Sprintf("user %s has id %s", u.ID, u.Name)
}

type SafeCounter struct {
	mu sync.Mutex
	n  int
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func WithRequestTimeout(parent context.Context) context.Context {
	ctx, _ := context.WithTimeout(parent, 2*time.Second)
	return ctx
}
