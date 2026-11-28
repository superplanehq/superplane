package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomAutomationIntegrationStartMode marks a custom automation trigger that
// the user chose from a connected integration.
const CustomAutomationIntegrationStartMode = "integration"

// CurrentStageKey is the board column of the task, or the custom stage that
// its latest dispatch is in. The board column shows custom stages as Implement.
func (o *FactoryWorkOrder) CurrentStageKey(tx *gorm.DB) (string, error) {
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

// PullRequestMergeHandedOff reports whether a stage automation took over the
// move of the task after the specified pull request merged.
func (o *FactoryWorkOrder) PullRequestMergeHandedOff(tx *gorm.DB, pullRequestID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&FactoryPullRequest{}).
		Where("id = ? AND work_order_id = ? AND state = ? AND merge_handoff_canvas_id IS NOT NULL", pullRequestID, o.ID, FactoryPullRequestStateMerged).
		Count(&count).Error
	return count > 0, err
}

// RecordPullRequestMergeHandoff marks a merged pull request of the task as
// handled by the stage automation in canvasID.
func (o *FactoryWorkOrder) RecordPullRequestMergeHandoff(tx *gorm.DB, pullRequestID, canvasID uuid.UUID) error {
	return tx.Model(&FactoryPullRequest{}).
		Where("id = ? AND work_order_id = ?", pullRequestID, o.ID).
		Update("merge_handoff_canvas_id", canvasID).Error
}
