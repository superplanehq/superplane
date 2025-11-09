package workflows

import (
    "fmt"

    "github.com/superplanehq/superplane/pkg/models"
)

// expandBlueprintNodes takes top-level workflow nodes and returns an expanded list including
// internal nodes from referenced blueprints. Internal nodes are namespaced as
// "<parentNodeID>:<internalNodeID>" and tagged with metadata {"internal": true}.
func expandBlueprintNodes(organizationID string, nodes []models.Node) ([]models.Node, error) {
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

        // Expand first-level internal nodes. Nested blueprints will be represented
        // as internal nodes as well (without recursively expanding for now).
        for _, bn := range b.Nodes {
            internal := models.Node{
                ID:            n.ID + ":" + bn.ID,
                Name:          bn.Name,
                Type:          bn.Type,
                Ref:           bn.Ref,
                Configuration: bn.Configuration,
                Metadata:      cloneMetadataWithInternal(bn.Metadata),
                Position:      bn.Position,
                IsCollapsed:   bn.IsCollapsed,
            }
            expanded = append(expanded, internal)
        }
    }

    return expanded, nil
}

func cloneMetadataWithInternal(md map[string]any) map[string]any {
    out := map[string]any{}
    for k, v := range md {
        out[k] = v
    }
    out["internal"] = true
    return out
}

