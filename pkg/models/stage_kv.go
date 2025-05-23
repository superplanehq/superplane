package models

import (
	"time"

	uuid "github.com/google/uuid"
)

type StageKV struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	StageID   uuid.UUID
	Key       string
	Value     string
	CreatedAt *time.Time
}
