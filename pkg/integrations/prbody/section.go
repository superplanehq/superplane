package prbody

import (
	"fmt"
	"regexp"
	"strings"
)

var bodySectionKey = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

func ValidateBodySectionKey(key string) error {
	if len(key) > 64 || !bodySectionKey.MatchString(key) {
		return fmt.Errorf("sectionKey must use lowercase letters, numbers, and single hyphens, starting with a letter (maximum 64 characters)")
	}
	return nil
}

// UpdateBodySection preserves every byte outside one ordered marker pair.
// Rejected boundaries leave the original description intact.
func UpdateBodySection(description, key, body string) (string, error) {
	if err := ValidateBodySectionKey(key); err != nil {
		return description, err
	}
	startMarker := "<!-- superplane:" + key + ":start -->"
	endMarker := "<!-- superplane:" + key + ":end -->"
	if strings.Contains(body, startMarker) || strings.Contains(body, endMarker) {
		return description, fmt.Errorf("section body contains its own boundary markers")
	}
	section := startMarker + "\n" + body + "\n" + endMarker
	starts, ends := strings.Count(description, startMarker), strings.Count(description, endMarker)
	if starts == 0 && ends == 0 {
		if description == "" {
			return section, nil
		}
		return description + "\n\n" + section, nil
	}
	start, end := strings.Index(description, startMarker), strings.Index(description, endMarker)
	if starts != 1 || ends != 1 || end < start {
		return description, fmt.Errorf("section markers are incomplete, reversed, or duplicated")
	}
	return description[:start] + section + description[end+len(endMarker):], nil
}
