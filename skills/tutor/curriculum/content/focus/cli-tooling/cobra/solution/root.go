package main

import (
	"encoding/json"
	"fmt"
	"io"

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
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
		// Execute still returns the error; these two only stop cobra from
		// printing it, plus a wall of usage text, behind main's back.
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().String("format", "text", `output format: "text" or "json"`)
	root.AddCommand(newAddCmd(app), newListCmd(app), newTagCmd(app))
	return root
}

// helpRunE makes a grouping command runnable. Without a Run/RunE, cobra decides
// the command needs help *before* it validates arguments, so `notes tag bogus`
// would print help and exit 0 instead of reporting the typo.
func helpRunE(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

func newAddCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := app.Resolve(cmd)
			if err != nil {
				return err
			}
			tags, err := cmd.Flags().GetStringSlice("tag")
			if err != nil {
				return err
			}
			note := app.Store.Add(args[0], tags)
			if set.Format == "json" {
				return writeJSON(cmd.OutOrStdout(), note)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "added %s\n", note.ID)
			return err
		},
	}
	cmd.Flags().StringSlice("tag", nil, "tag to attach (repeatable)")
	return cmd
}

func newListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			set, err := app.Resolve(cmd)
			if err != nil {
				return err
			}
			notes := app.Store.All()
			if set.Limit > 0 && len(notes) > set.Limit {
				notes = notes[:set.Limit]
			}
			out := cmd.OutOrStdout()
			if set.Format == "json" {
				return writeJSON(out, struct {
					Notes []Note `json:"notes"`
				}{Notes: notes})
			}
			if len(notes) == 0 {
				_, err := fmt.Fprintln(out, "no notes")
				return err
			}
			for _, n := range notes {
				if _, err := fmt.Fprintln(out, noteLine(n)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().Int("limit", 0, "show at most N notes (0 means all)")
	return cmd
}

func newTagCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Work with tags",
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
	}
	cmd.AddCommand(newTagAddCmd(app), newTagListCmd(app))
	return cmd
}

func newTagAddCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "add <note-id> <tag>...",
		Short: "Attach tags to a note",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := app.Resolve(cmd)
			if err != nil {
				return err
			}
			note, err := app.Store.Tag(args[0], args[1:]...)
			if err != nil {
				return err
			}
			if set.Format == "json" {
				return writeJSON(cmd.OutOrStdout(), note)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "tagged %s\n", note.ID)
			return err
		},
	}
}

func newTagListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every tag in use",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			set, err := app.Resolve(cmd)
			if err != nil {
				return err
			}
			tags := app.Store.Tags()
			out := cmd.OutOrStdout()
			if set.Format == "json" {
				return writeJSON(out, struct {
					Tags []string `json:"tags"`
				}{Tags: tags})
			}
			if len(tags) == 0 {
				_, err := fmt.Fprintln(out, "no tags")
				return err
			}
			for _, t := range tags {
				if _, err := fmt.Fprintln(out, t); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func noteLine(n Note) string {
	line := n.ID + " " + n.Text
	for _, t := range n.Tags {
		line += " #" + t
	}
	return line
}

func writeJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
