package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
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
	WorkOrderID uuid.UUID `gorm:"primaryKey"`
	UserID      uuid.UUID `gorm:"primaryKey"`
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

func (o *FactoryWorkOrder) UpdateAssignees(tx *gorm.DB, assigneeIDs []uuid.UUID, updatedBy uuid.UUID) error {
	previousAssignees := o.AssigneeIDs()

	if err := o.ReplaceAssignees(tx, assigneeIDs); err != nil {
		return err
	}

	now := time.Now()
	o.UpdatedAt = now
	if err := tx.Model(o).Update("updated_at", now).Error; err != nil {
		return err
	}

	assigned, unassigned := assigneeDiff(previousAssignees, assigneeIDs)
	return o.RecordAssigneesUpdated(tx, updatedBy, assigned, unassigned)
}

func (o *FactoryWorkOrder) Close(db *gorm.DB, result string, closedBy *uuid.UUID) (*FactoryWorkOrder, error) {
	if o.State == FactoryWorkOrderStateClosed {
		return o, nil
	}

	now := time.Now()

	err := db.Transaction(func(tx *gorm.DB) error {
		o.State = FactoryWorkOrderStateClosed
		o.Result = result
		o.UpdatedAt = now

		updateErr := tx.
			Model(o).
			Updates(map[string]any{
				"state":      o.State,
				"result":     o.Result,
				"updated_at": o.UpdatedAt,
			}).
			Error

		if updateErr != nil {
			return updateErr
		}

		return o.RecordClosed(tx, closedBy, result)
	})

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

func (o *FactoryWorkOrder) ListEvents(tx *gorm.DB, limit int, before *time.Time) ([]FactoryWorkOrderEvent, error) {
	query := tx.
		Where("organization_id = ? AND factory_id = ? AND work_order_id = ?", o.OrganizationID, o.FactoryID, o.ID)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	var events []FactoryWorkOrderEvent
	err := query.
		Order("created_at DESC").
		Order("id DESC").
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (o *FactoryWorkOrder) CountEvents(tx *gorm.DB) (int64, error) {
	var count int64

	err := tx.
		Model(&FactoryWorkOrderEvent{}).
		Where("organization_id = ? AND factory_id = ? AND work_order_id = ?", o.OrganizationID, o.FactoryID, o.ID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (o *FactoryWorkOrder) Ref() *factory.WorkOrderRef {
	return &factory.WorkOrderRef{
		ID:    o.ID,
		Title: o.Title,
	}
}

func (o *FactoryWorkOrder) RecordOpened(tx *gorm.DB, createdBy *uuid.UUID) error {
	data := factory.WorkOrderOpened{
		Order: o.Ref(),
	}
	if createdBy != nil {
		data.User = &factory.UserRef{ID: *createdBy}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Type:           factory.EventTypeOrderOpened,
		Data:           datatypes.JSON(jsonData),
		CreatedAt:      time.Now(),
	}

	return tx.Create(event).Error
}

func (o *FactoryWorkOrder) RecordClosed(tx *gorm.DB, closedBy *uuid.UUID, result string) error {
	data := factory.WorkOrderClosed{
		Order:  o.Ref(),
		User:   &factory.UserRef{ID: *closedBy},
		Result: &result,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Type:           factory.EventTypeOrderClosed,
		Data:           datatypes.JSON(jsonData),
		CreatedAt:      time.Now(),
	}

	return tx.Create(event).Error
}

func (o *FactoryWorkOrder) RecordAssigneesUpdated(
	tx *gorm.DB,
	updatedBy uuid.UUID,
	assigned []factory.UserRef,
	unassigned []factory.UserRef,
) error {
	if len(assigned) == 0 && len(unassigned) == 0 {
		return nil
	}

	data := factory.WorkOrderAssigneesUpdated{
		Order:      o.Ref(),
		User:       &factory.UserRef{ID: updatedBy},
		Assigned:   assigned,
		Unassigned: unassigned,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Type:           factory.EventTypeOrderAssigneesUpdated,
		Data:           datatypes.JSON(jsonData),
		CreatedAt:      time.Now(),
	}

	return tx.Create(event).Error
}

func (o *FactoryWorkOrder) AssigneeIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(o.Assignees))
	for _, assignee := range o.Assignees {
		ids = append(ids, assignee.UserID)
	}

	return ids
}

func assigneeDiff(previousIDs, nextIDs []uuid.UUID) (assigned, unassigned []factory.UserRef) {
	previous := make(map[uuid.UUID]struct{}, len(previousIDs))
	for _, id := range previousIDs {
		previous[id] = struct{}{}
	}

	next := make(map[uuid.UUID]struct{}, len(nextIDs))
	for _, id := range nextIDs {
		next[id] = struct{}{}
	}

	for id := range next {
		if _, ok := previous[id]; !ok {
			assigned = append(assigned, factory.UserRef{ID: id})
		}
	}

	for id := range previous {
		if _, ok := next[id]; !ok {
			unassigned = append(unassigned, factory.UserRef{ID: id})
		}
	}

	return assigned, unassigned
}
