package models

import (
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomAutomationIntegrationStartMode marks a custom automation trigger that
// the user chose from a connected integration.
const CustomAutomationIntegrationStartMode = "integration"

// StageHandlesPullRequestMerge reports whether a custom automation in the
// current stage of the task listens for merged pull requests. That automation
// moves the task, so PR Closure must not close the task on a merge.
func (o *FactoryWorkOrder) StageHandlesPullRequestMerge(tx *gorm.DB) (bool, error) {
	stage, err := o.currentStageKey(tx)
	if err != nil || stage == "" {
		return false, err
	}
	factory, err := FindFactory(tx, o.OrganizationID, o.FactoryID)
	if err != nil {
		return false, err
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return false, err
	}
	ids := make([]uuid.UUID, 0, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID != nil && canvases[i].ColumnKey != nil && *canvases[i].ColumnKey == stage {
			ids = append(ids, canvases[i].ID)
		}
	}
	specs, err := FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return false, err
	}
	for _, spec := range specs {
		if FactoryAppTemplateID(spec.Nodes) != FactoryAppTemplateCustomAutomationID {
			continue
		}
		if slices.ContainsFunc(spec.Nodes, nodeListensForPullRequestMerge) {
			return true, nil
		}
	}
	return false, nil
}

// currentStageKey is the board column of the task, or the custom stage that
// its latest dispatch is in. The board column shows custom stages as Implement.
func (o *FactoryWorkOrder) currentStageKey(tx *gorm.DB) (string, error) {
	if o.State == FactoryWorkOrderStateOpen {
		grouped, err := ListWorkOrderLineDispatchesByWorkOrderIDs(tx, []uuid.UUID{o.ID})
		if err != nil {
			return "", err
		}
		dispatches := grouped[o.ID]
		if n := len(dispatches); n > 0 && strings.HasPrefix(dispatches[n-1].CurrentStageKey, "custom-") {
			return dispatches[n-1].CurrentStageKey, nil
		}
	}
	return WorkOrderBoardColumn(tx, o)
}

func nodeListensForPullRequestMerge(node Node) bool {
	template, _ := node.Metadata[FactoryAppTemplateMetadataKey].(map[string]any)
	if mode, _ := template["startMode"].(string); node.Type != NodeTypeTrigger || mode != CustomAutomationIntegrationStartMode {
		return false
	}
	actions, _ := node.Configuration["actions"].([]any)
	switch node.ComponentName() {
	case "github.onPullRequest":
		return slices.Contains(actions, any("closed"))
	case "bitbucket.onPullRequest":
		return slices.Contains(actions, any("merged"))
	}
	return false
}
