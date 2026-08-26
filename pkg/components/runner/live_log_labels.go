package runner

import "strings"

const (
	LiveLogKindBash       = "bash"
	LiveLogKindPrompt     = "prompt"
	LiveLogKindSetup      = "setup"
	LiveLogKindJavaScript = "javascript"
	LiveLogKindPython     = "python"

	liveLogPreviewMaxRunes = 2048
)

// LiveLogPreview is the first non-empty line of user-facing command or prompt text.
// The log row ellipsizes in the UI. The rune cap only bounds a huge one-line prompt.
func LiveLogPreview(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > liveLogPreviewMaxRunes {
			return string(runes[:liveLogPreviewMaxRunes])
		}
		return line
	}
	return ""
}
