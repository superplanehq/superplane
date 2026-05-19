package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	TaskStatusQueued     = "queued"
	TaskStatusDispatched = "dispatched"
	TaskStatusRunning    = "running"
	TaskStatusSucceeded  = "succeeded"
	TaskStatusFailed     = "failed"
	TaskStatusCanceled   = "canceled"
)

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
