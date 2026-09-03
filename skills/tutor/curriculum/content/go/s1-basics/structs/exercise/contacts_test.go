package main

import "testing"

func TestNewContact(t *testing.T) {
	c := NewContact("Ada Lovelace", 36, "ada@example.com", "555-0100")
	if c.Person.Name != "Ada Lovelace" {
		t.Errorf("c.Person.Name = %q, want %q (Name belongs in the embedded Person)",
			c.Person.Name, "Ada Lovelace")
	}
	if c.Name != "Ada Lovelace" {
		t.Errorf("c.Name (promoted) = %q, want %q", c.Name, "Ada Lovelace")
	}
	if c.Age != 36 {
		t.Errorf("c.Age = %d, want %d", c.Age, 36)
	}
	if c.Email != "ada@example.com" {
		t.Errorf("c.Email = %q, want %q", c.Email, "ada@example.com")
	}
	if c.Phone != "555-0100" {
		t.Errorf("c.Phone = %q, want %q", c.Phone, "555-0100")
	}
}

func TestRename(t *testing.T) {
	orig := NewContact("Ada Lovelace", 36, "ada@example.com", "555-0100")
	if orig.Name != "Ada Lovelace" {
		t.Fatalf("NewContact isn't done yet (Name = %q) — make TestNewContact pass first", orig.Name)
	}
	renamed := Rename(orig, "Ada King")
	if renamed.Name != "Ada King" {
		t.Errorf("Rename(orig, %q).Name = %q, want %q", "Ada King", renamed.Name, "Ada King")
	}
	if renamed.Age != orig.Age || renamed.Email != orig.Email || renamed.Phone != orig.Phone {
		t.Errorf("Rename changed more than the name: got %+v", renamed)
	}
	if orig.Name != "Ada Lovelace" {
		t.Errorf("Rename modified the caller's contact: orig.Name = %q — return a changed copy instead",
			orig.Name)
	}
}

func TestSameContact(t *testing.T) {
	ada := NewContact("Ada Lovelace", 36, "ada@example.com", "555-0100")
	cases := []struct {
		name string
		a, b Contact
		want bool
	}{
		{"same data built separately", ada,
			NewContact("Ada Lovelace", 36, "ada@example.com", "555-0100"), true},
		{"different name", ada,
			NewContact("Grace Hopper", 36, "ada@example.com", "555-0100"), false},
		{"different age", ada,
			NewContact("Ada Lovelace", 37, "ada@example.com", "555-0100"), false},
		{"different phone", ada,
			NewContact("Ada Lovelace", 36, "ada@example.com", "555-0199"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SameContact(c.a, c.b); got != c.want {
				t.Errorf("SameContact(%+v, %+v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestFindByEmail(t *testing.T) {
	book := []Contact{
		NewContact("Ada Lovelace", 36, "ada@example.com", "555-0100"),
		NewContact("Grace Hopper", 45, "grace@example.com", "555-0199"),
	}

	c, ok := FindByEmail(book, "grace@example.com")
	if !ok {
		t.Fatalf("FindByEmail(book, %q): ok = false, want true", "grace@example.com")
	}
	if c.Name != "Grace Hopper" {
		t.Errorf("FindByEmail(book, %q).Name = %q, want %q",
			"grace@example.com", c.Name, "Grace Hopper")
	}

	c, ok = FindByEmail(book, "nobody@example.com")
	if ok {
		t.Errorf("FindByEmail(book, %q): ok = true, want false", "nobody@example.com")
	}
	if c != (Contact{}) {
		t.Errorf("a miss should return the zero Contact, got %+v", c)
	}
}

func TestSameGroup(t *testing.T) {
	cases := []struct {
		name string
		a, b Group
		want bool
	}{
		{"same name and members",
			Group{Name: "book club", Members: []string{"ada", "grace"}},
			Group{Name: "book club", Members: []string{"ada", "grace"}}, true},
		{"both member lists empty",
			Group{Name: "new"},
			Group{Name: "new"}, true},
		{"different name",
			Group{Name: "book club", Members: []string{"ada"}},
			Group{Name: "chess club", Members: []string{"ada"}}, false},
		{"different member",
			Group{Name: "book club", Members: []string{"ada", "grace"}},
			Group{Name: "book club", Members: []string{"ada", "linus"}}, false},
		{"different member count",
			Group{Name: "book club", Members: []string{"ada", "grace"}},
			Group{Name: "book club", Members: []string{"ada"}}, false},
		{"same members, different order",
			Group{Name: "book club", Members: []string{"ada", "grace"}},
			Group{Name: "book club", Members: []string{"grace", "ada"}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SameGroup(c.a, c.b); got != c.want {
				t.Errorf("SameGroup(%+v, %+v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
