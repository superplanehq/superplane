package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CanvasMemory struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	CanvasID  uuid.UUID
	Namespace string
	Values    datatypes.JSONType[any]
}

func (CanvasMemory) TableName() string {
	return "canvas_memories"
}

func AddCanvasMemoryInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, values any) error {
	record := CanvasMemory{
		CanvasID:  canvasID,
		Namespace: namespace,
		Values:    datatypes.NewJSONType(values),
	}

	return tx.Create(&record).Error
}

func AddCanvasMemory(canvasID uuid.UUID, namespace string, values any) error {
	return AddCanvasMemoryInTransaction(database.Conn(), canvasID, namespace, values)
}

func ListCanvasMemoriesInTransaction(tx *gorm.DB, canvasID uuid.UUID) ([]CanvasMemory, error) {
	var records []CanvasMemory
	err := tx.
		Where("canvas_id = ?", canvasID).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func ListCanvasMemories(canvasID uuid.UUID) ([]CanvasMemory, error) {
	return ListCanvasMemoriesInTransaction(database.Conn(), canvasID)
}

func DeleteCanvasMemory(canvasID, memoryID uuid.UUID) error {
	return DeleteCanvasMemoryInTransaction(database.Conn(), canvasID, memoryID)
}

func DeleteCanvasMemoryInTransaction(tx *gorm.DB, canvasID, memoryID uuid.UUID) error {
	return tx.
		Where("canvas_id = ? AND id = ?", canvasID, memoryID).
		Delete(&CanvasMemory{}).
		Error
}
