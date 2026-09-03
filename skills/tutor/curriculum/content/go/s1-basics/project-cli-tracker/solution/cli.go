package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tutor.local/project-cli-tracker/tracker"
)

// dispatch runs one subcommand against t and returns the text main should
// print. args is os.Args without the program name, e.g. ["add", "buy",
// "milk"]. Supported subcommands (exact outputs in LESSON.md):
//
//	add <words...>   words joined with spaces become the title
//	list             one line per task, or "no tasks yet"
//	complete <id>    marks the task done
//	summary          "N open, M done"
//
// No arguments, an unknown subcommand, or bad subcommand input is an
// error. Errors from the tracker are returned as-is or wrapped — never
// swallowed.
func dispatch(t *tracker.Tracker, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("usage: tracker <add|list|complete|summary>")
	}
	switch args[0] {
	case "add":
		task, err := t.Add(strings.Join(args[1:], " "))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("added #%d: %s", task.ID, task.Title), nil
	case "list":
		tasks := t.List()
		if len(tasks) == 0 {
			return "no tasks yet", nil
		}
		lines := make([]string, 0, len(tasks))
		for _, task := range tasks {
			box := "[ ]"
			if task.Done {
				box = "[x]"
			}
			lines = append(lines, fmt.Sprintf("%s #%d %s", box, task.ID, task.Title))
		}
		return strings.Join(lines, "\n"), nil
	case "complete":
		if len(args) != 2 {
			return "", errors.New("usage: tracker complete <id>")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return "", fmt.Errorf("complete: %q is not a task id", args[1])
		}
		if err := t.Complete(id); err != nil {
			return "", err
		}
		return fmt.Sprintf("completed #%d", id), nil
	case "summary":
		counts := t.Summary()
		return fmt.Sprintf("%d open, %d done", counts["open"], counts["done"]), nil
	default:
		return "", fmt.Errorf("unknown subcommand %q (want add, list, complete, or summary)", args[0])
	}
}
