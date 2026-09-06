package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"tutor.local/snippets/store"
)

const usage = `usage: snippets [-path file] <command>

commands:
  add <name> <text>   store a snippet under a name
  get <name>          print one snippet
  list                print all snippet names
`

func main() {
	path := flag.String("path", "snippets.json", "file the snippets are stored in")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	s, err := store.Open(*path)
	if err != nil {
		fatal(err)
	}

	switch args[0] {
	case "add":
		if len(args) != 3 {
			fatal(errors.New("add needs a name and a text"))
		}
		if err := s.Add(args[1], args[2]); err != nil {
			fatal(err)
		}
	case "get":
		if len(args) != 2 {
			fatal(errors.New("get needs a name"))
		}
		text, err := s.Get(args[1])
		if err != nil {
			fatal(err)
		}
		fmt.Println(text)
	case "list":
		for _, name := range s.List() {
			fmt.Println(name)
		}
	default:
		fatal(fmt.Errorf("unknown command %q", args[0]))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "snippets:", err)
	os.Exit(1)
}
