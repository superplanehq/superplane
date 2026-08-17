package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FactoryWorkOrderQueueItem is a work order waiting for admission into a
// line step that is at its maxParallelism. No run or execution exists for
// it yet; both are created when the step admits the work order, and the
// queue item is deleted.
type FactoryWorkOrderQueueItem struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	LineID         uuid.UUID
	StepIndex      int
	StepName       string
	CreatedAt      time.Time
}

func (i *FactoryWorkOrderQueueItem) Delete(tx *gorm.DB) error {
	return tx.Delete(i).Error
}

// FactoryWorkOrderQueueItemRecord is a queue item joined with its line and
// 1-based position inside the step's queue, for API serialization.
type FactoryWorkOrderQueueItemRecord struct {
	FactoryWorkOrderQueueItem
	LineName string
	Position int
	// LineSteps snapshot the containing line's step definitions so the UI
	// can render the step sequence without a separate lookup.
	LineSteps datatypes.JSONSlice[FactoryLineStep] `gorm:"column:line_steps"`
}

func ListFactoryWorkOrderQueueItemsByWorkOrderIDs(
	tx *gorm.DB,
	workOrderIDs []uuid.UUID,
) (map[uuid.UUID][]FactoryWorkOrderQueueItemRecord, error) {
	result := make(map[uuid.UUID][]FactoryWorkOrderQueueItemRecord, len(workOrderIDs))
	if len(workOrderIDs) == 0 {
		return result, nil
	}

	var records []FactoryWorkOrderQueueItemRecord
	err := tx.
		Table("factory_work_order_queue_items AS q").
		Select(`
			q.*,
			l.name AS line_name,
			l.steps AS line_steps,
			(
				SELECT COUNT(*)
				FROM factory_work_order_queue_items ahead
				WHERE ahead.line_id = q.line_id
				AND ahead.step_index = q.step_index
				AND (ahead.created_at, ahead.id) < (q.created_at, q.id)
			) + 1 AS position
		`).
		Joins("JOIN factory_lines l ON l.id = q.line_id").
		Where("q.work_order_id IN ?", workOrderIDs).
		Order("q.created_at ASC").
		Order("q.id ASC").
		Scan(&records).
		Error
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		result[record.WorkOrderID] = append(result[record.WorkOrderID], record)
	}

	return result, nil
}

func findOldestFactoryWorkOrderQueueItem(tx *gorm.DB, lineID uuid.UUID, stepIndex int) (*FactoryWorkOrderQueueItem, error) {
	var item FactoryWorkOrderQueueItem
	err := tx.
		Where("line_id = ? AND step_index = ?", lineID, stepIndex).
		Order("created_at ASC").
		Order("id ASC").
		First(&item).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}
