package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

type WorkflowInitialEvent struct {
	ID         uuid.UUID
	WorkflowID uuid.UUID
	Data       datatypes.JSONType[map[string]any]
	CreatedAt  *time.Time
}

func FindWorkflowInitialEvent(id uuid.UUID) (*WorkflowInitialEvent, error) {
	var event WorkflowInitialEvent
	err := database.Conn().
		Where("id = ?", id).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}
