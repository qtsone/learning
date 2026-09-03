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
	// TODO: return a Contact built with a field-named literal.
	// Name and Age belong to the embedded Person, so the literal
	// needs a Person: Person{…} entry.
	return Contact{}
}

// Rename returns a copy of c with its name replaced. The caller's
// original is untouched — structs are values.
func Rename(c Contact, newName string) Contact {
	// TODO: set the (promoted) Name field on c, then return c.
	return c
}

// SameContact reports whether a and b hold exactly the same data.
func SameContact(a, b Contact) bool {
	// TODO: one comparison is enough — every Contact field is comparable.
	return false
}

// FindByEmail returns the contact with the given email and true, or the
// zero Contact and false when no entry matches.
func FindByEmail(book []Contact, email string) (Contact, bool) {
	// TODO: loop over book and compare each contact's Email.
	return Contact{}, false
}

// SameGroup reports whether a and b have the same name and the same
// members in the same order. == on Group does not compile, so compare
// field by field.
func SameGroup(a, b Group) bool {
	// TODO: compare the names, then the member counts, then each
	// member pairwise.
	return false
}
