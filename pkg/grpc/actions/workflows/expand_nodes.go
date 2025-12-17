package workflows

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
)

/*
 * Expand nodes takes top-level workflow nodes and returns an expanded list including
 * internal nodes from referenced blueprints. Internal nodes are namespaced as
 * "<parentNodeID>:<internalNodeID>".
 */
func expandNodes(organizationID string, nodes []models.Node) ([]models.Node, error) {
	expanded := make([]models.Node, 0, len(nodes))

	for _, n := range nodes {
		expanded = append(expanded, n)

		if n.Type != models.NodeTypeBlueprint || n.Ref.Blueprint == nil {
			continue
		}

		blueprintID := n.Ref.Blueprint.ID
		if blueprintID == "" {
			return nil, fmt.Errorf("blueprint node %s missing blueprint id", n.ID)
		}

		b, err := models.FindBlueprint(organizationID, blueprintID)
		if err != nil {
			return nil, fmt.Errorf("blueprint %s not found: %w", blueprintID, err)
		}

		for _, bn := range b.Nodes {
			internal := models.Node{
				ID:                n.ID + ":" + bn.ID,
				Name:              bn.Name,
				Type:              bn.Type,
				Ref:               bn.Ref,
				Configuration:     bn.Configuration,
				Metadata:          cloneMetadata(bn.Metadata),
				Position:          bn.Position,
				IsCollapsed:       bn.IsCollapsed,
				AppInstallationID: bn.AppInstallationID,
			}

			expanded = append(expanded, internal)
		}
	}

	return expanded, nil
}

func cloneMetadata(md map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range md {
		out[k] = v
	}
	return out
}
