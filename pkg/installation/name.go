package installation

import (
	"strings"
	"unicode"
)

const maxInstallationNameLength = 50

// DefaultInstallationName derives a human-readable app name from a GitHub repository name.
// Example: preview-env-github-digitalocean -> Preview Env Github Digitalocean
func DefaultInstallationName(repoName string) string {
	name := humanizeRepoName(repoName)
	if name == "" {
		return "Untitled App"
	}

	return truncateInstallationName(name)
}

func truncateInstallationName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	if len(name) <= maxInstallationNameLength {
		return name
	}

	return strings.TrimSpace(name[:maxInstallationNameLength])
}

func humanizeRepoName(repoName string) string {
	trimmed := strings.TrimSpace(repoName)
	if trimmed == "" {
		return ""
	}

	segments := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})

	words := make([]string, 0, len(segments))
	for _, segment := range segments {
		word := titleWord(segment)
		if word != "" {
			words = append(words, word)
		}
	}

	return strings.Join(words, " ")
}

func titleWord(word string) string {
	word = strings.TrimSpace(word)
	if word == "" {
		return ""
	}

	runes := []rune(strings.ToLower(word))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
