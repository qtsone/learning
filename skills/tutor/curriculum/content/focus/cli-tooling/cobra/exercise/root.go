package main

import (
	"github.com/spf13/cobra"
)

// NewRootCmd builds a fresh command tree bound to app.
//
// Fresh matters: a cobra.Command stores parsed flag values on itself, so a tree
// shared between runs remembers the previous run's flags. Building the tree in
// a constructor — never in a package-level var plus init() — keeps every run
// independent.
func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "notes",
		Short: "Keep short notes with tags",
	}
	// TODO: silence cobra's own error and usage printing so the value returned
	// by Execute is the single channel through which failure travels.
	// TODO: declare the persistent --format flag (default "text") on root.
	// TODO: attach the add, list and tag subcommands.
	return root
}

// TODO: newAddCmd(app), newListCmd(app) and newTagCmd(app) — the last one is a
// group command with its own add and list children.
