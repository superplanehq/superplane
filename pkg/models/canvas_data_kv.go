package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CanvasDataKV struct {
	ID         uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID uuid.UUID
	Key        string
	Value      datatypes.JSONType[any]
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

func (CanvasDataKV) TableName() string {
	return "workflow_data_kvs"
}

func UpsertCanvasDataKVInTransaction(tx *gorm.DB, workflowID uuid.UUID, key string, value any) error {
	now := time.Now()
	record := CanvasDataKV{
		WorkflowID: workflowID,
		Key:        key,
		Value:      datatypes.NewJSONType(value),
		UpdatedAt:  &now,
	}

	return tx.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "workflow_id"},
				{Name: "key"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"value":      datatypes.NewJSONType(value),
				"updated_at": &now,
			}),
		}).
		Create(&record).
		Error
}

func FindCanvasDataKVInTransaction(tx *gorm.DB, workflowID uuid.UUID, key string) (*CanvasDataKV, error) {
	var record CanvasDataKV
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("key = ?", key).
		First(&record).
		Error
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func ListCanvasDataKVsInTransaction(tx *gorm.DB, workflowID uuid.UUID) ([]CanvasDataKV, error) {
	var records []CanvasDataKV
	err := tx.
		Where("workflow_id = ?", workflowID).
		Order("key ASC").
		Find(&records).
		Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func UpsertCanvasDataKV(workflowID uuid.UUID, key string, value any) error {
	return UpsertCanvasDataKVInTransaction(database.Conn(), workflowID, key, value)
}

func FindCanvasDataKV(workflowID uuid.UUID, key string) (*CanvasDataKV, error) {
	return FindCanvasDataKVInTransaction(database.Conn(), workflowID, key)
}

func ListCanvasDataKVs(workflowID uuid.UUID) ([]CanvasDataKV, error) {
	return ListCanvasDataKVsInTransaction(database.Conn(), workflowID)
}
