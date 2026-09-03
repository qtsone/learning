package auth

import (
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewSessionIDIsUnpredictable(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		id, err := NewSessionID()
		if err != nil {
			t.Fatalf("NewSessionID() = error %v, want nil", err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(id)
		if err != nil {
			t.Fatalf("NewSessionID() = %q, which is not unpadded base64url: %v", id, err)
		}
		if len(raw) < SessionIDBytes {
			t.Fatalf("NewSessionID() carries %d random bytes, want at least %d", len(raw), SessionIDBytes)
		}
		if seen[id] {
			t.Fatalf("NewSessionID() repeated %q after %d calls", id, i)
		}
		seen[id] = true
	}
}

func TestSessionStoreEnforcesExpiryOnRead(t *testing.T) {
	clock := newFakeClock()
	store := NewSessionStore(clock, testTTL)

	sess, err := store.New("u1")
	if err != nil {
		t.Fatalf("New(\"u1\") = error %v, want nil", err)
	}
	if sess.ID == "" || sess.UserID != "u1" {
		t.Fatalf("New(\"u1\") = %+v, want a non-empty ID for user u1", sess)
	}
	if !sess.IssuedAt.Equal(testStart) {
		t.Errorf("IssuedAt = %v, want the injected clock's time %v", sess.IssuedAt, testStart)
	}
	if want := testStart.Add(testTTL); !sess.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v (now + the store's TTL)", sess.ExpiresAt, want)
	}
	if got, ok := store.Lookup(sess.ID); !ok || got.ID != sess.ID {
		t.Fatalf("Lookup(fresh id) = (%+v, %v), want the session and true", got, ok)
	}

	clock.Advance(testTTL - time.Nanosecond)
	if _, ok := store.Lookup(sess.ID); !ok {
		t.Errorf("Lookup() one nanosecond before expiry = false, want true")
	}

	clock.Advance(time.Nanosecond)
	if _, ok := store.Lookup(sess.ID); ok {
		t.Errorf("Lookup() at ExpiresAt = true, want false: the session has expired")
	}
	if n := store.Len(); n != 0 {
		t.Errorf("store holds %d session(s) after an expired lookup, want 0: expired sessions must be dropped", n)
	}
}

func TestRotate(t *testing.T) {
	clock := newFakeClock()
	store := NewSessionStore(clock, testTTL)
	sess, err := store.New("u1")
	if err != nil {
		t.Fatalf("New(\"u1\") = error %v, want nil", err)
	}

	clock.Advance(time.Minute)
	now := clock.Now()
	rotated, err := store.Rotate(sess.ID)
	if err != nil {
		t.Fatalf("Rotate(live id) = error %v, want nil", err)
	}
	if rotated.ID == sess.ID {
		t.Errorf("Rotate() kept the id %q; rotation must mint a new one", rotated.ID)
	}
	if rotated.UserID != "u1" {
		t.Errorf("Rotate() = user %q, want u1: the session keeps its owner", rotated.UserID)
	}
	if !rotated.IssuedAt.Equal(now) || !rotated.ExpiresAt.Equal(now.Add(testTTL)) {
		t.Errorf("Rotate() = issued %v expiring %v, want %v and %v", rotated.IssuedAt, rotated.ExpiresAt, now, now.Add(testTTL))
	}
	if _, ok := store.Lookup(sess.ID); ok {
		t.Errorf("the old id still works after Rotate(); it must be destroyed")
	}
	if _, ok := store.Lookup(rotated.ID); !ok {
		t.Errorf("the rotated id does not work")
	}
	if n := store.Len(); n != 1 {
		t.Errorf("store holds %d session(s) after rotation, want 1", n)
	}

	if _, err := store.Rotate("no-such-id"); !errors.Is(err, ErrNoSession) {
		t.Errorf("Rotate(unknown id) = error %v, want ErrNoSession", err)
	}

	clock.Advance(testTTL)
	if _, err := store.Rotate(rotated.ID); !errors.Is(err, ErrNoSession) {
		t.Errorf("Rotate(expired id) = error %v, want ErrNoSession", err)
	}
}

func TestSessionStoreIsSafeForConcurrentUse(t *testing.T) {
	store := NewSessionStore(newFakeClock(), testTTL)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				sess, err := store.New("u1")
				if err != nil {
					t.Errorf("New() = error %v, want nil", err)
					return
				}
				if _, ok := store.Lookup(sess.ID); !ok {
					t.Errorf("Lookup(just-created id) = false, want true")
					return
				}
				store.Delete(sess.ID)
			}
		}()
	}
	wg.Wait()

	if n := store.Len(); n != 0 {
		t.Errorf("store holds %d session(s) after every goroutine deleted its own, want 0", n)
	}
}
