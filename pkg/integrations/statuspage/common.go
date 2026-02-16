package statuspage

import (
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

// resolveComponentNameOrIDs fetches page components and resolves names to IDs.
// The Statuspage API expects component IDs (e.g. "8kbf7d35c070"), not names.
// Values in nameOrIDToStatus can be either component IDs or component names.
// Returns (componentIDs, componentIDToStatus, error).
func resolveComponentNameOrIDs(client *Client, pageID string, nameOrIDToStatus map[string]string) ([]string, map[string]string, error) {
	comps, err := client.ListComponents(pageID)
	if err != nil {
		return nil, nil, err
	}
	nameOrIDToResolvedID := make(map[string]string)
	for _, c := range comps {
		nameOrIDToResolvedID[c.Name] = c.ID
		nameOrIDToResolvedID[c.ID] = c.ID
	}
	ids := make([]string, 0, len(nameOrIDToStatus))
	statusByID := make(map[string]string)
	seen := make(map[string]bool)
	for nameOrID, status := range nameOrIDToStatus {
		resolved := nameOrIDToResolvedID[nameOrID]
		if resolved == "" {
			resolved = nameOrID
		}
		if !seen[resolved] {
			ids = append(ids, resolved)
			seen[resolved] = true
		}
		statusByID[resolved] = status
	}
	return ids, statusByID, nil
}

// extractComponentIDs returns component IDs from config.
// Supports format: components = [ { componentId, status }, ... ]
func extractComponentIDs(config map[string]any) []string {
	v, ok := config["components"]
	if !ok || v == nil {
		return nil
	}
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["componentId"].(string)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
