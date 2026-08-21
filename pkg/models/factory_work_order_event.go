package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type FactoryWorkOrderEvent struct {
	ID          uuid.UUID
	WorkOrderID uuid.UUID
	Type        string
	Data        datatypes.JSON
	CreatedAt   time.Time
}
