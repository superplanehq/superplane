package gitlab

import (
	"fmt"
	"slices"
)

// changedListField handles both list-diff shapes GitLab sends: the common {previous, current}
// object (labels, assignees) and the bare [previous, current] pair GitLab uses for reviewers.
func changedListField(changes map[string]any, field string) (previous, current []any, ok bool) {
	switch change := changes[field].(type) {
	case map[string]any:
		previous, _ = change["previous"].([]any)
		current, _ = change["current"].([]any)
		return previous, current, true
	case []any:
		if len(change) != 2 {
			return nil, nil, false
		}
		previous, _ = change[0].([]any)
		current, _ = change[1].([]any)
		return previous, current, true
	default:
		return nil, nil, false
	}
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

// changedToValue reports the field now holding a non-nil value that differs from its previous value.
func changedToValue(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	current, hasCurrent := change["current"]
	return hasCurrent && current != nil && current != change["previous"]
}

// changedToNil reports the field being cleared from a non-nil previous value.
func changedToNil(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	current, hasCurrent := change["current"]
	return hasCurrent && current == nil && change["previous"] != nil
}

// changedBoolTo reports current == target, but only if previous actually differed (guards against no-op entries).
func changedBoolTo(changes map[string]any, field string, target bool) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	current, ok := change["current"].(bool)
	if !ok || current != target {
		return false
	}

	previous, ok := change["previous"].(bool)
	return !ok || previous != target
}

// changedField reports the field being present in changes with a value that actually differs.
func changedField(changes map[string]any, field string) bool {
	change, ok := changes[field].(map[string]any)
	if !ok {
		return false
	}

	return change["previous"] != change["current"]
}
