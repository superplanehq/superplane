package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CanvasSubscription struct {
	SourceCanvasID uuid.UUID `gorm:"column:source_canvas_id;primaryKey"`
	TargetCanvasID uuid.UUID `gorm:"column:target_canvas_id;primaryKey"`
	TargetNodeID   string    `gorm:"column:target_node_id;primaryKey"`
}

func (c *CanvasSubscription) TableName() string {
	return "canvas_subscriptions"
}

func DeleteCanvasSubscriptionsForNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	return tx.
		Where("target_canvas_id = ? AND target_node_id = ?", workflowID, nodeID).
		Delete(&CanvasSubscription{}).
		Error
}
