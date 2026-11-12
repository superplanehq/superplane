// Package diagnostics prepares bounded runner failure messages for user reports.
package diagnostics

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var credentialPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:Bearer|Basic)\s+[^\s"']+`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
	regexp.MustCompile(`\b(?:sk|rk|gh[pousr]|github_pat|xox[baprs]|glpat|npm_)-?[A-Za-z0-9_-]{8,}\b`),
	regexp.MustCompile(`\b(?:AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_-]{35})\b`),
}
var urlCredentialPattern = regexp.MustCompile(`(?i)(https?://)[^/@\s]+:[^/@\s]+@`)
var assignmentPattern = regexp.MustCompile(`(?i)((?:[a-z0-9_-]*(?:api[_-]?key|token|secret|password|credential|signature)|sig)["']?\s*[=:]\s*["']?)[^\s&"']+`)
var ansiPattern = regexp.MustCompile("\x1b\\[[0-?]*[ -/]*[@-~]")

// Sanitize removes known environment values and common credential formats before
// applying the byte limit. Callers must supply values available to the process.
func Sanitize(text string, values []string, limit int) string {
	text = ansiPattern.ReplaceAllString(text, "")
	values = slices.Clone(values)
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	for _, value := range values {
		if value != "" {
			encoded, _ := json.Marshal(value)
			text = strings.ReplaceAll(text, string(encoded[1:len(encoded)-1]), "[REDACTED]")
			text = strings.ReplaceAll(text, value, "[REDACTED]")
		}
	}
	for _, pattern := range credentialPatterns {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}
	text = urlCredentialPattern.ReplaceAllString(text, "${1}[REDACTED]@")
	text = assignmentPattern.ReplaceAllString(text, "${1}[REDACTED]")
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
	text = strings.TrimSpace(text)
	if limit > 0 && len(text) > limit {
		if limit < len("\n[truncated]") {
			return ""
		}
		text = text[:limit-len("\n[truncated]")]
		for !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
		text += "\n[truncated]"
	}
	return text
}

// WithLogReference keeps the same reference in task, component, and run errors.
func WithLogReference(message, taskID string) string {
	if strings.TrimSpace(taskID) == "" {
		return message + "\nTask log reference was not recorded."
	}
	reference := "Logs: runner task " + taskID + "."
	if strings.HasSuffix(message, reference) {
		return message
	}
	return message + "\n" + reference
}
