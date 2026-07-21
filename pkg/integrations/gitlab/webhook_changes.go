package gitlab

import (
	"fmt"
	"slices"
)

// changedListField returns the previous/current lists for a `changes` entry
// whose value is a list (e.g. labels, assignees, reviewer_ids).
func changedListField(changes map[string]any, field string) (previous, current []any, ok bool) {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return nil, nil, false
	}

	previous, _ = change["previous"].([]any)
	current, _ = change["current"].([]any)
	return previous, current, true
}

// listItemKeys extracts a comparable key from each list item: the given
// object key (e.g. "id") for object items, or the item itself for scalars
// (e.g. reviewer_ids, a list of plain integers).
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

// listGrew reports whether `current` has an item not present in `previous`.
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

// listShrank reports whether `previous` has an item no longer present in `current`.
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

// changedFromNilToValue reports whether a scalar `changes` field went from
// unset to set (e.g. a milestone was assigned).
func changedFromNilToValue(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	current, hasCurrent := change["current"]
	return change["previous"] == nil && hasCurrent && current != nil
}

// changedFromValueToNil reports whether a scalar `changes` field went from
// set to unset (e.g. a milestone was removed).
func changedFromValueToNil(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	previous, hasPrevious := change["previous"]
	return hasPrevious && previous != nil && change["current"] == nil
}

// changedBoolTo reports whether a boolean `changes` field's current value equals target.
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

// changedField reports whether a `changes` field is present at all,
// regardless of its previous/current values (e.g. title, description).
func changedField(changes map[string]any, field string) bool {
	_, ok := changes[field]
	return ok
}
