package models

import (
	"sort"
	"strings"
	"unicode"
)

// CanonicalModelName removes a single provider prefix from a model name.
func CanonicalModelName(id string) string {
	name := strings.TrimSpace(id)
	if _, rest, found := strings.Cut(name, "/"); found && rest != "" && !strings.Contains(rest, "/") {
		return rest
	}
	return name
}

// ModelHasVersion reports whether a model name contains a version digit.
func ModelHasVersion(model string) bool {
	for _, r := range model {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// FirstVersionedModelID returns the latest matching model id in lexical order.
func FirstVersionedModelID(ids []string, alias string) string {
	needle := strings.ToLower(strings.TrimSpace(alias))
	if needle == "" {
		return ""
	}
	matches := make([]string, 0, len(ids))
	for _, id := range ids {
		normalized := strings.ToLower(strings.TrimSpace(id))
		if normalized == "" {
			continue
		}
		name := normalized
		if _, rest, found := strings.Cut(normalized, "/"); found && rest != "" {
			name = rest
		}
		if name == needle || !ModelHasVersion(name) {
			continue
		}
		if !modelNameContainsToken(name, needle) {
			continue
		}
		matches = append(matches, strings.TrimSpace(id))
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	return matches[len(matches)-1]
}

func modelNameContainsToken(name, token string) bool {
	for _, part := range strings.Split(name, "-") {
		if part == token {
			return true
		}
	}
	return false
}
