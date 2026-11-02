package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FactoryWorkOrderRefinementRun links one refinement run to its task.
// A planning session keeps only its current run. This table keeps every run,
// so earlier runs stay in the task history after a new chat message.
type FactoryWorkOrderRefinementRun struct {
	RunID       uuid.UUID `gorm:"primaryKey"`
	WorkOrderID uuid.UUID
	CreatedAt   time.Time
}

func (FactoryWorkOrderRefinementRun) TableName() string {
	return "factory_work_order_refinement_runs"
}

// RecordRefinementRun saves the durable link between a refinement run and the
// task. It is safe to call again for the same run.
func (o *FactoryWorkOrder) RecordRefinementRun(tx *gorm.DB, runID uuid.UUID) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&FactoryWorkOrderRefinementRun{
		RunID:       runID,
		WorkOrderID: o.ID,
		CreatedAt:   time.Now(),
	}).Error
}

// LatestBacklogRefinementRun also finds runs rejected before a planning session
// or durable task-run link exists.
func (o *FactoryWorkOrder) LatestBacklogRefinementRun(tx *gorm.DB, canvasIDs []uuid.UUID) (*CanvasRun, error) {
	if len(canvasIDs) == 0 {
		return nil, nil
	}
	var run CanvasRun
	result := tx.Raw(`
		SELECT r.* FROM workflow_runs r
		JOIN workflow_events e ON e.run_id = r.id
		WHERE r.workflow_id IN ? AND r.node_id = ?
		  AND e.execution_id IS NULL
		  AND e.data->>'type' = ?
		  AND e.data->'data'->'workOrder'->>'id' = ?
		ORDER BY r.created_at DESC, r.id DESC LIMIT 1
	`, canvasIDs, FactoryAppBacklogTriggerID, workOrderCreatedPayloadType, o.ID.String()).Scan(&run)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return &run, nil
}
