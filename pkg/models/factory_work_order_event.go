package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FactoryWorkOrderEvent struct {
	ID          uuid.UUID
	WorkOrderID uuid.UUID
	Type        string
	Content     datatypes.JSONType[map[string]any]
	CreatedAt   time.Time
}

func (FactoryWorkOrderEvent) TableName() string {
	return "factory_work_order_events"
}

func CreateFactoryWorkOrderEvent(
	tx *gorm.DB,
	workOrderID uuid.UUID,
	eventType string,
	content map[string]any,
) (*FactoryWorkOrderEvent, error) {
	if content == nil {
		content = map[string]any{}
	}

	event := &FactoryWorkOrderEvent{
		WorkOrderID: workOrderID,
		Type:        eventType,
		Content:     datatypes.NewJSONType(content),
		CreatedAt:   time.Now(),
	}

	if err := tx.Clauses(clause.Returning{}).Create(event).Error; err != nil {
		return nil, err
	}

	return event, nil
}

func ListFactoryWorkOrderEvents(tx *gorm.DB, workOrderID uuid.UUID) ([]FactoryWorkOrderEvent, error) {
	var events []FactoryWorkOrderEvent
	err := tx.
		Where("work_order_id = ?", workOrderID).
		Order("created_at DESC").
		Order("id DESC").
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	return events, nil
}
