package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	RunnerTaskBackendLegacy     = "legacy"
	RunnerTaskBackendIntegrated = "integrated"

	RunnerTaskStateQueued    = "queued"
	RunnerTaskStateReserved  = "reserved"
	RunnerTaskStateRunning   = "running"
	RunnerTaskStateSucceeded = "succeeded"
	RunnerTaskStateFailed    = "failed"
	RunnerTaskStateCanceled  = "canceled"
	RunnerTaskStateLost      = "lost"
)

var (
	ErrRunnerTaskNotFound        = errors.New("runner task not found")
	ErrRunnerTaskNotReservable   = errors.New("runner task is not available for reservation")
	ErrRunnerTaskFleetMismatch   = errors.New("runner task fleet does not match runner fleet")
	ErrRunnerTaskBackendMismatch = errors.New("runner task does not use the integrated backend")
	ErrRunnerTaskAlreadyAssigned = errors.New("runner task already has a runner")
)

/*
 * RunnerTask is the durable execution unit assigned to one fleet runner.
 * PayloadCiphertext protects task inputs, which can contain credentials and
 * other secrets. Result is visible workflow output. SuperPlane stores it
 * temporarily until workflow processing creates the corresponding event.
 */
type RunnerTask struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	FleetID           uuid.UUID
	RunnerID          *uuid.UUID
	Backend           string
	State             string
	PayloadCiphertext []byte
	Result            datatypes.JSON
	ExitCode          *int32
	CancelRequestedAt *time.Time
	QueuedAt          time.Time
	ReservedAt        *time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func FindRunnerTask(tx *gorm.DB, id uuid.UUID) (*RunnerTask, error) {
	var task RunnerTask
	err := tx.Where("id = ?", id).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (t *RunnerTask) Reserve(tx *gorm.DB, runnerID uuid.UUID) error {
	if t.State != RunnerTaskStateQueued || t.RunnerID != nil {
		return ErrRunnerTaskNotReservable
	}

	now := time.Now()
	result := tx.Model(t).
		Where("state = ? AND runner_id IS NULL", RunnerTaskStateQueued).
		Updates(map[string]any{
			"runner_id":   runnerID,
			"state":       RunnerTaskStateReserved,
			"reserved_at": now,
			"updated_at":  now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRunnerTaskNotReservable
	}

	t.RunnerID = &runnerID
	t.State = RunnerTaskStateReserved
	t.ReservedAt = &now
	t.UpdatedAt = now
	return nil
}

/*
 * TaskLogUpload tracks only an active task log upload.
 * It provides the next accepted chunk sequence and byte count while
 * SuperPlane receives or compacts chunks.
 * SuperPlane removes this transient row after it publishes the final compressed log object.
 */
type TaskLogUpload struct {
	TaskID            uuid.UUID
	NextChunkSequence int64
	TotalBytes        int64
	FinalizingAt      *time.Time
	UpdatedAt         time.Time
}

func (TaskLogUpload) TableName() string {
	return "runner_task_log_uploads"
}
