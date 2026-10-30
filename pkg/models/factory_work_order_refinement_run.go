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
