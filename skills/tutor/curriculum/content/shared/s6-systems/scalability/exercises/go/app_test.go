package scalability

import (
	"fmt"
	"sync"
	"testing"
)

func TestLoginWhoamiSameInstance(t *testing.T) {
	app := NewApp(NewMemoryStore())
	token := app.Login("ada")
	user, ok := app.Whoami(token)
	if !ok || user != "ada" {
		t.Fatalf("Whoami(token from own Login) = %q, %v; want %q, true", user, ok, "ada")
	}
}

func TestAnyInstanceCanServeAnyRequest(t *testing.T) {
	store := NewMemoryStore()
	a1 := NewApp(store)
	a2 := NewApp(store)

	token := a1.Login("ada")
	user, ok := a2.Whoami(token)
	if !ok || user != "ada" {
		t.Fatalf("instance 2 Whoami(token from instance 1) = %q, %v; want %q, true — "+
			"session state must live in the shared store, not on the instance", user, ok, "ada")
	}
}

func TestUnknownToken(t *testing.T) {
	app := NewApp(NewMemoryStore())
	if user, ok := app.Whoami("no-such-token"); ok {
		t.Fatalf("Whoami(unknown token) = %q, true; want ok=false", user)
	}
}

func TestConcurrentLoginsAcrossInstances(t *testing.T) {
	store := NewMemoryStore()
	apps := []*App{NewApp(store), NewApp(store)}
	tokens := make([][]string, len(apps))

	var wg sync.WaitGroup
	for i, app := range apps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				tokens[i] = append(tokens[i], app.Login(fmt.Sprintf("user-%d-%d", i, j)))
			}
		}()
	}
	wg.Wait()

	for i := range apps {
		other := apps[(i+1)%len(apps)]
		for j, token := range tokens[i] {
			want := fmt.Sprintf("user-%d-%d", i, j)
			if user, ok := other.Whoami(token); !ok || user != want {
				t.Fatalf("other instance Whoami(token %d/%d) = %q, %v; want %q, true", i, j, user, ok, want)
			}
		}
	}
}
