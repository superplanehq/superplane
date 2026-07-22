package gitlab

import (
	"fmt"
	"slices"
)

func changedListField(changes map[string]any, field string) (previous, current []any, ok bool) {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return nil, nil, false
	}

	previous, _ = change["previous"].([]any)
	current, _ = change["current"].([]any)
	return previous, current, true
}

// listItemKeys extracts the object key (e.g. "id") from each item, or the item itself for scalar lists.
func listItemKeys(items []any, key string) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			if v, ok := m[key]; ok {
				keys = append(keys, fmt.Sprintf("%v", v))
			}
			continue
		}
		keys = append(keys, fmt.Sprintf("%v", item))
	}
	return keys
}

func listGrew(changes map[string]any, field, idKey string) bool {
	previous, current, ok := changedListField(changes, field)
	if !ok {
		return false
	}

	previousKeys := listItemKeys(previous, idKey)
	for _, key := range listItemKeys(current, idKey) {
		if !slices.Contains(previousKeys, key) {
			return true
		}
	}

	return false
}

func listShrank(changes map[string]any, field, idKey string) bool {
	previous, current, ok := changedListField(changes, field)
	if !ok {
		return false
	}

	currentKeys := listItemKeys(current, idKey)
	for _, key := range listItemKeys(previous, idKey) {
		if !slices.Contains(currentKeys, key) {
			return true
		}
	}

	return false
}

func changedFromNilToValue(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	current, hasCurrent := change["current"]
	return change["previous"] == nil && hasCurrent && current != nil
}

func changedFromValueToNil(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	previous, hasPrevious := change["previous"]
	return hasPrevious && previous != nil && change["current"] == nil
}

func changedBoolTo(changes map[string]any, field string, target bool) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	current, ok := change["current"].(bool)
	if !ok {
		return false
	}

	return current == target
}

func changedField(changes map[string]any, field string) bool {
	_, ok := changes[field]
	return ok
}
