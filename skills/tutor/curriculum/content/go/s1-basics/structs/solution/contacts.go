package main

// Person holds the identity fields every address-book record shares.
type Person struct {
	Name string
	Age  int
}

// Contact is one address-book entry. Person is embedded, so its fields
// are promoted: c.Name and c.Age reach into the inner Person.
type Contact struct {
	Person
	Email string
	Phone string
}

// Group is a named set of member names. The slice field makes Group
// values impossible to compare with == (see LESSON.md).
type Group struct {
	Name    string
	Members []string
}

// NewContact builds a Contact from its parts.
func NewContact(name string, age int, email, phone string) Contact {
	return Contact{
		Person: Person{Name: name, Age: age},
		Email:  email,
		Phone:  phone,
	}
}

// Rename returns a copy of c with its name replaced. The caller's
// original is untouched — structs are values.
func Rename(c Contact, newName string) Contact {
	c.Name = newName
	return c
}

// SameContact reports whether a and b hold exactly the same data.
func SameContact(a, b Contact) bool {
	return a == b
}

// FindByEmail returns the contact with the given email and true, or the
// zero Contact and false when no entry matches.
func FindByEmail(book []Contact, email string) (Contact, bool) {
	for _, c := range book {
		if c.Email == email {
			return c, true
		}
	}
	return Contact{}, false
}

// SameGroup reports whether a and b have the same name and the same
// members in the same order. == on Group does not compile, so compare
// field by field.
func SameGroup(a, b Group) bool {
	if a.Name != b.Name || len(a.Members) != len(b.Members) {
		return false
	}
	for i, m := range a.Members {
		if m != b.Members[i] {
			return false
		}
	}
	return true
}
