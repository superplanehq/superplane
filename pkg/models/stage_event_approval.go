package models

import (
	"time"

	uuid "github.com/google/uuid"
)

type StageEventApproval struct {
	ID           uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	StageEventID uuid.UUID
	ApprovedAt   *time.Time
	ApprovedBy   *uuid.UUID
}
