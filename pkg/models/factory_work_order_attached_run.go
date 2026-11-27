package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FactoryWorkOrderAttachedRun links a run with no task, such as a webhook
// run, to a task that waited in the run's stage. The task timeline shows the
// run in that stage.
type FactoryWorkOrderAttachedRun struct {
	WorkOrderID uuid.UUID `gorm:"primaryKey"`
	RunID       uuid.UUID `gorm:"primaryKey"`
	StageKey    string
	CreatedAt   time.Time
}

func (FactoryWorkOrderAttachedRun) TableName() string {
	return "factory_work_order_attached_runs"
}

// AttachWaitingRun links runID to the task. The task must wait in stageKey.
// It is safe to call again for the same run.
func (o *FactoryWorkOrder) AttachWaitingRun(tx *gorm.DB, runID uuid.UUID, stageKey string) error {
	if err := lockRunningCaller(tx, runID); err != nil {
		return err
	}
	if _, err := o.lockWaitingLineDispatch(tx, stageKey); err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&FactoryWorkOrderAttachedRun{
		WorkOrderID: o.ID,
		RunID:       runID,
		StageKey:    stageKey,
		CreatedAt:   time.Now(),
	}).Error
}

// ListWorkOrdersAttachedToRun returns the tasks that show runID on their timeline.
func ListWorkOrdersAttachedToRun(tx *gorm.DB, runID uuid.UUID) ([]FactoryWorkOrder, error) {
	var orders []FactoryWorkOrder
	err := tx.Joins("JOIN factory_work_order_attached_runs a ON a.work_order_id = factory_work_orders.id").
		Where("a.run_id = ?", runID).
		Find(&orders).Error
	return orders, err
}

type FactoryWorkOrderAttachedRunRecord struct {
	RunID      uuid.UUID
	StageKey   string
	CanvasID   uuid.UUID
	CanvasName string
	RunState   string
	RunResult  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt *time.Time
}

func ListWorkOrderAttachedRuns(tx *gorm.DB, workOrderID uuid.UUID) ([]FactoryWorkOrderAttachedRunRecord, error) {
	var records []FactoryWorkOrderAttachedRunRecord
	err := tx.Raw(`
		SELECT a.run_id, a.stage_key, r.workflow_id AS canvas_id, w.name AS canvas_name,
		       r.state AS run_state, COALESCE(r.result, '') AS run_result,
		       r.created_at, r.updated_at, r.finished_at
		FROM factory_work_order_attached_runs a
		JOIN workflow_runs r ON r.id = a.run_id
		JOIN workflows w ON w.id = r.workflow_id
		WHERE a.work_order_id = ?
		ORDER BY a.created_at, a.run_id
	`, workOrderID).Scan(&records).Error
	return records, err
}
