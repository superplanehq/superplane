package statuspage

import (
	"encoding/json"
	"strings"
)

// NodeMetadata contains metadata stored on component nodes for display in the UI.
type NodeMetadata struct {
	PageName       string   `json:"pageName"`
	ComponentNames []string `json:"componentNames,omitempty"`
}

// containsExpression returns true if any string in the slice contains an expression placeholder.
func containsExpression(ids []string) bool {
	for _, id := range ids {
		if strings.Contains(id, "{{") {
			return true
		}
	}
	return false
}

// extractComponentIDs returns component IDs from config, handling various formats the frontend may send.
func extractComponentIDs(config map[string]any) []string {
	v, ok := config["componentIds"]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		ids := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				ids = append(ids, s)
			}
		}
		return ids
	case string:
		var ids []string
		if err := json.Unmarshal([]byte(val), &ids); err == nil {
			return ids
		}
		return nil
	default:
		return nil
	}
}
