package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CanvasMemory struct {
	ID        uuid.UUID
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
	memories := []CanvasMemory{}
	err := tx.
		Where("canvas_id = ?", canvasID).
		Order("namespace ASC").
		Find(&memories).
		Error
	return memories, err
}

func ListCanvasMemories(canvasID uuid.UUID) ([]CanvasMemory, error) {
	return ListCanvasMemoriesInTransaction(database.Conn(), canvasID)
}

func DeleteCanvasMemoryInTransaction(tx *gorm.DB, canvasID uuid.UUID, memoryID uuid.UUID) error {
	return tx.
		Where("id = ? AND canvas_id = ?", memoryID, canvasID).
		Delete(&CanvasMemory{}).
		Error
}

func DeleteCanvasMemory(canvasID uuid.UUID, memoryID uuid.UUID) error {
	return DeleteCanvasMemoryInTransaction(database.Conn(), canvasID, memoryID)
}
