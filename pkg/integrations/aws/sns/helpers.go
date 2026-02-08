package sns

import (
	"fmt"
	"sort"
	"strings"
)

// topicNameFromArn extracts the topic name from a topic ARN.
func topicNameFromArn(topicArn string) string {
	parts := strings.Split(strings.TrimSpace(topicArn), ":")
	if len(parts) == 0 {
		return strings.TrimSpace(topicArn)
	}

	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return strings.TrimSpace(topicArn)
	}

	return name
}

// mapAnyToStringMap converts object-like configuration values to string maps.
func mapAnyToStringMap(raw map[string]any) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	var keys []string
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if raw[key] == nil {
			continue
		}

		normalized := strings.TrimSpace(fmt.Sprint(raw[key]))
		if normalized == "" {
			continue
		}
		values[key] = normalized
	}

	if len(values) == 0 {
		return nil
	}

	return values
}

// boolAttribute returns true when the attribute exists and equals "true" (case-insensitive).
func boolAttribute(attributes map[string]string, key string) bool {
	value, ok := attributes[key]
	if !ok {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(value), "true")
}
