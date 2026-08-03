package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	FactoryWorkOrderStateOpen   = "open"
	FactoryWorkOrderStateClosed = "closed"

	FactoryWorkOrderResultCompleted = "completed"
	FactoryWorkOrderResultRejected  = "rejected"
)

var ErrFactoryWorkOrderNotFound = errors.New("factory work order not found")

type FactoryWorkOrder struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Title          string
	Description    string
	State          string
	Result         string
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time

	CreatedBy *User                      `gorm:"foreignKey:CreatedByID"`
	Assignees []FactoryWorkOrderAssignee `gorm:"foreignKey:WorkOrderID"`
}

func (FactoryWorkOrder) TableName() string {
	return "factory_work_orders"
}

func (o *FactoryWorkOrder) IsOpen() bool {
	return o.State == FactoryWorkOrderStateOpen
}

type FactoryWorkOrderAssignee struct {
	WorkOrderID uuid.UUID
	UserID      uuid.UUID
	CreatedAt   time.Time

	User *User `gorm:"foreignKey:UserID"`
}

func (FactoryWorkOrderAssignee) TableName() string {
	return "factory_work_order_assignees"
}

// NOTE: this is only OK to be used in the workers.
// For APIs, always use the factory.FindWorkOrder method.
func FindUnscopedWorkOrder(db *gorm.DB, id uuid.UUID) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := db.Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (o *FactoryWorkOrder) UpdateAssignees(tx *gorm.DB, assigneeIDs []uuid.UUID) (*FactoryWorkOrder, error) {
	now := time.Now()
	if err := o.ReplaceAssignees(tx, assigneeIDs); err != nil {
		return nil, err
	}

	o.UpdatedAt = now
	if err := tx.Model(o).Update("updated_at", now).Error; err != nil {
		return nil, err
	}

	return o, nil
}

func (o *FactoryWorkOrder) Close(tx *gorm.DB, result string) (*FactoryWorkOrder, error) {
	now := time.Now()
	o.State = FactoryWorkOrderStateClosed
	o.Result = result
	o.UpdatedAt = now
	err := tx.Model(o).
		Where("organization_id = ? AND factory_id = ? AND id = ?", o.OrganizationID, o.FactoryID, o.ID).
		Where("state = ?", FactoryWorkOrderStateOpen).
		Updates(o).Error

	if err != nil {
		return nil, err
	}

	return o, nil
}

func (o *FactoryWorkOrder) FindActiveExecution(tx *gorm.DB) (*FactoryWorkOrderExecution, error) {
	var execution FactoryWorkOrderExecution
	err := tx.
		Where("work_order_id = ?", o.ID).
		Where("status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func (o *FactoryWorkOrder) ReplaceAssignees(tx *gorm.DB, assigneeIDs []uuid.UUID) error {
	if err := tx.Where("work_order_id = ?", o.ID).Delete(&FactoryWorkOrderAssignee{}).Error; err != nil {
		return err
	}

	if len(assigneeIDs) == 0 {
		return nil
	}

	now := time.Now()
	assignees := make([]FactoryWorkOrderAssignee, 0, len(assigneeIDs))
	for _, assigneeID := range assigneeIDs {
		assignees = append(assignees, FactoryWorkOrderAssignee{
			WorkOrderID: o.ID,
			UserID:      assigneeID,
			CreatedAt:   now,
		})
	}

	return tx.Create(&assignees).Error
}
