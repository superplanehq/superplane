package statuspage

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
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

// resolveMetadataSetup fetches page and component names from the API when pageID and componentIDs
// are static (no expressions). Returns empty metadata if HTTP context is unavailable or on error.
func resolveMetadataSetup(ctx core.SetupContext, pageID string, componentIDs []string) NodeMetadata {
	metadata := NodeMetadata{}
	if pageID == "" || strings.Contains(pageID, "{{") || ctx.HTTP == nil {
		return metadata
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return metadata
	}
	pages, err := client.ListPages()
	if err == nil {
		for _, p := range pages {
			if p.ID == pageID {
				metadata.PageName = p.Name
				break
			}
		}
	}
	if len(componentIDs) > 0 && !containsExpression(componentIDs) {
		components, err := client.ListComponents(pageID)
		if err == nil {
			idToName := make(map[string]string)
			for _, c := range components {
				idToName[c.ID] = c.Name
			}
			for _, id := range componentIDs {
				if name := idToName[id]; name != "" {
					metadata.ComponentNames = append(metadata.ComponentNames, name)
				} else {
					metadata.ComponentNames = append(metadata.ComponentNames, id)
				}
			}
		}
	}
	return metadata
}

// componentIDsForMetadataSetup returns component IDs from config or from getIDsFromSpec when config has none.
func componentIDsForMetadataSetup(config any, getIDsFromSpec func() []string) []string {
	configMap, _ := config.(map[string]any)
	if configMap == nil {
		configMap = make(map[string]any)
	}
	ids := extractComponentIDs(configMap)
	if len(ids) == 0 {
		return getIDsFromSpec()
	}
	return ids
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
