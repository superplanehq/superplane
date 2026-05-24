package store

import (
	"sort"
	"strings"
)

// NormalizeLabels trims, deduplicates, and sorts fleet labels.
func NormalizeLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(labels))
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// LabelsSubset reports whether have contains every label in required.
func LabelsSubset(have, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(have))
	for _, l := range have {
		set[l] = struct{}{}
	}
	for _, l := range required {
		if _, ok := set[l]; !ok {
			return false
		}
	}
	return true
}
