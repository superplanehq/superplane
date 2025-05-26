package models

import (
	"time"

	uuid "github.com/google/uuid"
)

type StageExecutionLabel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;"`
	SourceID    uuid.UUID
	SourceType  string
	ExecutionID uuid.UUID
	Name        string
	Value       string
	CreatedAt   *time.Time
}
