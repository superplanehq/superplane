package agent

import (
	"errors"
	"strings"

	"github.com/superplane/runner/shared/models"
)

// shellDirective is one command_list entry with optional display name for live logs.
type shellDirective struct {
	Text  string
	Shell string
}

func normalizeDirectiveLines(directives []string) []string {
	var out []string
	for _, s := range directives {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func directivesFromStrings(lines []string) []shellDirective {
	normalized := normalizeDirectiveLines(lines)
	out := make([]shellDirective, 0, len(normalized))
	for _, line := range normalized {
		out = append(out, shellDirective{Text: line, Shell: line})
	}
	return out
}

func directivesFromCommands(commands models.CommandList) []shellDirective {
	out := make([]shellDirective, 0, len(commands))
	for _, spec := range commands {
		shell := spec.ShellLine()
		if shell == "" {
			continue
		}
		out = append(out, shellDirective{
			Text:  spec.DisplayText(),
			Shell: shell,
		})
	}
	return out
}

func errEmptyCommands() error {
	return errors.New("empty commands")
}

// wrapSourcedDirective prepares a command_list entry for `source` in the shared
// interactive PTY shell.
//
// A top-level `exit` in user code would otherwise terminate that shell before the
// runner can write its end marker. Aliasing `exit` to `return` turns those into
// a normal non-zero source status while keeping cwd, exports, and background jobs.
func wrapSourcedDirective(shell string) string {
	var b strings.Builder
	b.WriteString("shopt -s expand_aliases\n")
	b.WriteString("alias exit=return\n")
	b.WriteString(strings.TrimRight(shell, "\n"))
	b.WriteString("\n")
	return b.String()
}
