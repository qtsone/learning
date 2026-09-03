package main

import "fmt"

func main() {
	book := []Contact{
		NewContact("Ada Lovelace", 36, "ada@example.com", "555-0100"),
		NewContact("Grace Hopper", 45, "grace@example.com", "555-0199"),
	}
	if c, ok := FindByEmail(book, "ada@example.com"); ok {
		fmt.Printf("found: %+v\n", c)
	} else {
		fmt.Println("nothing found — implement the TODOs in contacts.go")
	}
}
