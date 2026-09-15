package pricebook

import (
	"strings"
)

// ModelMatch is one catalog key that prices a model id.
type ModelMatch struct {
	Key  string
	Mode string
}

// NormalizeModelID lowercases a model id and strips a single provider prefix.
func NormalizeModelID(model string) string {
	return normalizeModelID(model)
}

// MatchModel finds the longest prefix, then a family token, for a model id.
func MatchModel(model string, prefixes []PrefixRate, families []FamilyRate) (ModelMatch, bool) {
	normalized := normalizeModelID(model)
	if key, ok := longestPrefixKey(normalized, prefixes); ok {
		return ModelMatch{Key: key, Mode: "prefix"}, true
	}
	if key, ok := firstFamilyKey(normalized, families); ok {
		return ModelMatch{Key: key, Mode: "family"}, true
	}
	return ModelMatch{}, false
}

func longestPrefixKey(normalized string, prefixes []PrefixRate) (string, bool) {
	bestPrefix := ""
	found := false
	for _, item := range prefixes {
		prefix := strings.ToLower(strings.TrimSpace(item.Prefix))
		if prefix == "" || !strings.HasPrefix(normalized, prefix) {
			continue
		}
		if !found || len(prefix) > len(bestPrefix) {
			bestPrefix = prefix
			found = true
		}
	}
	return bestPrefix, found
}

func firstFamilyKey(normalized string, families []FamilyRate) (string, bool) {
	parts := strings.Split(normalized, "-")
	for _, family := range families {
		token := strings.ToLower(strings.TrimSpace(family.Token))
		if token == "" {
			continue
		}
		for _, part := range parts {
			if part == token {
				return token, true
			}
		}
	}
	return "", false
}
