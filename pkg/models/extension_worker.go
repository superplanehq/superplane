package models

import (
	"time"

	"github.com/google/uuid"
)

type ExtensionWorker struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	State          string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

type ExtensionJob struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	ExtensionID    uuid.UUID
	VersionID      uuid.UUID
	State          string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}
