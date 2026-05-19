package runners

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type RunnerFleet struct {
	ID        uuid.UUID                    `gorm:"primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string                       `json:"name"`
	Mode      string                       `gorm:"not null;default:bridge" json:"mode"`
	FleetURL  string                       `json:"fleet_url,omitempty"`
	AuthToken string                       `json:"-"`
	Labels    datatypes.JSONType[[]string] `json:"labels"`
	CreatedAt *time.Time                   `json:"created_at"`
}

func (RunnerFleet) TableName() string { return "runner_fleets" }

func (f *RunnerFleet) UsesBridge() bool {
	return f.Mode == "" || f.Mode == FleetModeBridge
}

type RunnerTask struct {
	ID           uuid.UUID                   `gorm:"primaryKey;default:uuid_generate_v4()"`
	FleetID      uuid.UUID                   `gorm:"not null"`
	FleetTaskID  string                      `gorm:"not null"`
	ExecutionID  uuid.UUID                   `gorm:"not null"`
	Status       string                      `gorm:"not null;default:queued"`
	Spec         datatypes.JSONType[JobSpec] `gorm:"not null"`
	ExitCode     *int
	Output       string `gorm:"not null;default:''"`
	Error        string `gorm:"not null;default:''"`
	Result       datatypes.JSON
	TaskLog      datatypes.JSONType[*TaskLogSink]
	DispatchedAt *time.Time
	CompletedAt  *time.Time
	CreatedAt    *time.Time
}

func (RunnerTask) TableName() string { return "runner_tasks" }

func (t *RunnerTask) IsTerminal() bool {
	switch t.Status {
	case TaskStatusSucceeded, TaskStatusFailed, TaskStatusCanceled:
		return true
	default:
		return false
	}
}
