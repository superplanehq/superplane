package runner

import "strings"

const (
	LiveLogKindBash       = "bash"
	LiveLogKindPrompt     = "prompt"
	LiveLogKindSetup      = "setup"
	LiveLogKindJavaScript = "javascript"
	LiveLogKindPython     = "python"

	// liveLogTextMaxRunes bounds the size of the full command/prompt text we
	// send as the live-log preview, so a huge prompt cannot blow up payload
	// or DOM size.
	liveLogTextMaxRunes = 8192
)

// LiveLogText returns the full user-facing command or prompt text, with
// interior newlines preserved and leading/trailing blank lines trimmed. The
// Automations tab renders this as the step title, wrapped and with newlines
// preserved, so callers that need a one-line label should use
// LiveLogFirstLine instead.
func LiveLogText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > liveLogTextMaxRunes {
		return string(runes[:liveLogTextMaxRunes])
	}
	return trimmed
}

// LiveLogFirstLine returns the first non-empty line of text, for compact
// contexts (the runner live-log dialog header, CLI logs) that must stay
// single-line.
func LiveLogFirstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return line
	}
	return ""
}
