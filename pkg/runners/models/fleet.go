package models

import (
	"time"

	"github.com/google/uuid"
)

type RunnerFleet struct {
	ID        uuid.UUID  `gorm:"primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string     `json:"name"`
	AuthToken string     `json:"-"`
	CreatedAt *time.Time `json:"created_at"`
}

func (RunnerFleet) TableName() string { return "runner_fleets" }
