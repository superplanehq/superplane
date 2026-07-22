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

// ptyExitAliasBootstrap configures the shared interactive PTY shell once per
// session so top-level `exit` in sourced command_list entries becomes `return`.
// That keeps the shell alive for end markers while preserving cwd, exports, and
// background jobs. Applied at session boot, not before every command.
const ptyExitAliasBootstrap = `shopt -s expand_aliases; alias exit=return`

// wrapSourcedDirective enables fail-fast for a command_list entry without
// letting `set -e` kill the shared interactive PTY shell.
//
// An ERR trap converts the first failing command into `return` from the sourced
// script so the runner still gets an end-marker status. A RETURN trap always
// clears errexit/ERR afterwards — important when user code (or alias exit→return)
// returns early and would otherwise skip the trailing cleanup lines.
func wrapSourcedDirective(shell string) string {
	var b strings.Builder
	b.WriteString("set +e\n")
	b.WriteString("trap 'trap - ERR RETURN; set +e' RETURN\n")
	b.WriteString("trap '_sp_runner_err=$?; trap - ERR; set +e; return \"$_sp_runner_err\"' ERR\n")
	b.WriteString("set -e\n")
	b.WriteString(strings.TrimRight(shell, "\n"))
	b.WriteString("\n")
	b.WriteString("trap - ERR RETURN\n")
	b.WriteString("set +e\n")
	return b.String()
}
