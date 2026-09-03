package sq

import "testing"

func TestStackLIFO(t *testing.T) {
	var s Stack
	for _, r := range "abc" {
		s.Push(r)
	}
	want := []rune{'c', 'b', 'a'}
	for i, w := range want {
		got, ok := s.Pop()
		if !ok {
			t.Fatalf("Pop #%d: stack reported empty, want %q", i+1, w)
		}
		if got != w {
			t.Errorf("Pop #%d = %q, want %q (last in must come out first)", i+1, got, w)
		}
	}
}

func TestStackInterleaved(t *testing.T) {
	var s Stack
	s.Push('a')
	s.Push('b')
	if got, _ := s.Pop(); got != 'b' {
		t.Errorf("Pop after pushing a, b = %q, want 'b'", got)
	}
	s.Push('c')
	if got, _ := s.Pop(); got != 'c' {
		t.Errorf("Pop after pushing c = %q, want 'c'", got)
	}
	if got, _ := s.Pop(); got != 'a' {
		t.Errorf("final Pop = %q, want 'a'", got)
	}
}

func TestStackEmpty(t *testing.T) {
	var s Stack
	if v, ok := s.Pop(); ok {
		t.Errorf("Pop on empty stack = (%q, true), want ok = false", v)
	}
	if v, ok := s.Peek(); ok {
		t.Errorf("Peek on empty stack = (%q, true), want ok = false", v)
	}
}

func TestStackPeekDoesNotRemove(t *testing.T) {
	var s Stack
	s.Push('x')
	s.Push('y')
	for i := 1; i <= 2; i++ {
		got, ok := s.Peek()
		if !ok || got != 'y' {
			t.Fatalf("Peek call #%d = (%q, %v), want ('y', true)", i, got, ok)
		}
	}
	if s.Len() != 2 {
		t.Errorf("Len after two Peeks = %d, want 2 (Peek must not remove)", s.Len())
	}
}

func TestStackLen(t *testing.T) {
	var s Stack
	if s.Len() != 0 {
		t.Errorf("Len of new stack = %d, want 0", s.Len())
	}
	s.Push('a')
	s.Push('b')
	if s.Len() != 2 {
		t.Errorf("Len after two pushes = %d, want 2", s.Len())
	}
	s.Pop()
	if s.Len() != 1 {
		t.Errorf("Len after one pop = %d, want 1", s.Len())
	}
}
