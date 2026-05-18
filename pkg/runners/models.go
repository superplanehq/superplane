package runners

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type RunnerFleet struct {
	ID        uuid.UUID                    `gorm:"primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string                       `json:"name"`
	FleetURL  string                       `json:"fleet_url"`
	AuthToken string                       `json:"-"`
	Labels    datatypes.JSONType[[]string] `json:"labels"`
	CreatedAt *time.Time                   `json:"created_at"`
}

func (RunnerFleet) TableName() string { return "runner_fleets" }

type RunnerTask struct {
	ID          uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	FleetID     uuid.UUID `gorm:"not null"`
	FleetTaskID string    `gorm:"not null"`
	ExecutionID uuid.UUID `gorm:"not null"`
	CreatedAt   *time.Time
}

func (RunnerTask) TableName() string { return "runner_tasks" }
