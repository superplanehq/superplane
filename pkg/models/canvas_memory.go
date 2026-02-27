package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CanvasMemory struct {
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
