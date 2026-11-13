// Package diagnostics prepares bounded runner failure messages for user reports.
package diagnostics

import (
	"encoding/json"
	"net/url"
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
var queryParameterPattern = regexp.MustCompile(`[?&][^=\s&#"']+=[^\s&#"']+`)
var queryCredentialNamePattern = regexp.MustCompile(`(?i)(?:key|token|secret|password|credential|signature|sig)$`)
var jsonStringPattern = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)
var assignmentPattern = regexp.MustCompile(`(?i)((?:[a-z0-9_-]*(?:api[_-]?key|token|secret|password|credential|signature)|sig)["']?\s*[=:]\s*)("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s&"']+)`)
var ansiPattern = regexp.MustCompile("\x1b\\[[0-?]*[ -/]*[@-~]")

// Sanitize removes known environment values and common credential formats before
// applying the byte limit. Callers must supply values available to the process.
func Sanitize(text string, values []string, limit int) string {
	text = ansiPattern.ReplaceAllString(text, "")
	values = slices.Clone(values)
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	// Decode each JSON string, including strings embedded in ordinary log lines.
	text = jsonStringPattern.ReplaceAllStringFunc(text, func(encoded string) string {
		var decoded string
		if json.Unmarshal([]byte(encoded), &decoded) != nil {
			return encoded
		}
		redacted := redactCredentials(decoded, values)
		if redacted == decoded {
			return encoded
		}
		var output strings.Builder
		encoder := json.NewEncoder(&output)
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(redacted)
		return strings.TrimSuffix(output.String(), "\n")
	})
	text = redactCredentials(text, values)
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

func redactCredentials(text string, values []string) string {
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
	text = queryParameterPattern.ReplaceAllStringFunc(text, func(parameter string) string {
		prefix, _, _ := strings.Cut(parameter, "=")
		name, err := url.QueryUnescape(prefix[1:])
		if err != nil || !queryCredentialNamePattern.MatchString(name) {
			return parameter
		}
		return prefix + "=[REDACTED]"
	})
	return assignmentPattern.ReplaceAllStringFunc(text, func(assignment string) string {
		parts := assignmentPattern.FindStringSubmatch(assignment)
		value := "[REDACTED]"
		if parts[2][0] == '"' || parts[2][0] == '\'' {
			quote := parts[2][:1]
			value = quote + value + quote
		}
		return parts[1] + value
	})
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
