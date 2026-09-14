package models

import (
	"fmt"
	"maps"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	FactoryAppTemplateMetadataKey = "factoryTemplate"
	FactoryAppTemplateBacklogID   = "backlog"
	FactoryAppBacklogTriggerID    = "trigger"
)

// FactoryAppTemplateMetadata returns the persisted marker for a generated
// factory app. The marker survives canvas renames and user-facing copy changes.
func FactoryAppTemplateMetadata(templateID string, version int) map[string]any {
	return map[string]any{
		FactoryAppTemplateMetadataKey: map[string]any{
			"id":      templateID,
			"version": version,
		},
	}
}

func FactoryAppTemplateID(nodes []Node) string {
	for _, node := range nodes {
		metadata, ok := node.Metadata[FactoryAppTemplateMetadataKey].(map[string]any)
		if !ok {
			continue
		}
		if id, ok := metadata["id"].(string); ok {
			return id
		}
	}
	return ""
}

func IsBacklogFactoryApp(nodes []Node, _ []Edge) bool {
	if FactoryAppTemplateID(nodes) == FactoryAppTemplateBacklogID {
		return true
	}

	legacyNodeIDs := map[string]bool{
		FactoryAppBacklogTriggerID: true,
		"analyze":                  true,
		"report-confidence":        true,
		"attach-intent":            true,
		"add-run-error":            true,
	}
	if matchesBacklogIdentity(nodes, legacyNodeIDs) {
		return true
	}

	versionTwoNodeIDs := map[string]bool{
		FactoryAppBacklogTriggerID: true,
		"task-refinement-enabled":  true,
		"analyze":                  true,
		"refine-task":              true,
		"report-confidence":        true,
		"attach-intent":            true,
		"add-run-error":            true,
	}
	return matchesBacklogIdentity(nodes, versionTwoNodeIDs)
}

// StampFactoryAppTemplate records template identity in the live version and
// its normalized trigger node. Canvas changesets keep component metadata
// private, so generated templates stamp this server-owned key after publish.
func (c *Canvas) StampFactoryAppTemplate(tx *gorm.DB, triggerNodeID, templateID string, version int) error {
	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, c.ID)
	if err != nil {
		return err
	}

	metadata := FactoryAppTemplateMetadata(templateID, version)
	found := false
	for i := range liveVersion.Nodes {
		if liveVersion.Nodes[i].ID != triggerNodeID {
			continue
		}
		found = true
		liveVersion.Nodes[i].Metadata = maps.Clone(liveVersion.Nodes[i].Metadata)
		if liveVersion.Nodes[i].Metadata == nil {
			liveVersion.Nodes[i].Metadata = map[string]any{}
		}
		maps.Copy(liveVersion.Nodes[i].Metadata, metadata)
		break
	}
	if !found {
		return fmt.Errorf("template trigger node %s not found", triggerNodeID)
	}

	activeNode, err := FindCanvasNode(tx, c.ID, triggerNodeID)
	if err != nil {
		return err
	}
	activeMetadata := maps.Clone(activeNode.Metadata.Data())
	if activeMetadata == nil {
		activeMetadata = map[string]any{}
	}
	maps.Copy(activeMetadata, metadata)
	if err := tx.Model(activeNode).Update("metadata", activeMetadata).Error; err != nil {
		return err
	}

	return tx.Model(liveVersion).Update("nodes", datatypes.NewJSONSlice(liveVersion.Nodes)).Error
}

func matchesBacklogIdentity(nodes []Node, requiredNodeIDs map[string]bool) bool {
	if len(nodes) != len(requiredNodeIDs) {
		return false
	}

	analysisComponents := map[string]bool{
		SuperPlaneRunnerComponent: true,
		"runnerClaudeCode":        true,
		"runnerCodex":             true,
		"runnerOpenRouter":        true,
	}
	for _, node := range nodes {
		if !requiredNodeIDs[node.ID] {
			return false
		}
		if node.ID == "analyze" || node.ID == "refine-task" {
			if !analysisComponents[node.ComponentName()] {
				return false
			}
			continue
		}
		if node.ID == FactoryAppBacklogTriggerID && node.ComponentName() != "onWorkOrder" {
			return false
		}
	}
	return true
}
