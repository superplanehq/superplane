package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	ErrRunnerTaskNotFound           = errors.New("runner task not found")
	ErrRunnerTaskNotReservable      = errors.New("runner task is not available for reservation")
	ErrRunnerTaskFleetMismatch      = errors.New("runner task fleet does not match runner fleet")
	ErrRunnerTaskBackendMismatch    = errors.New("runner task does not use the integrated backend")
	ErrRunnerTaskAlreadyAssigned    = errors.New("runner task already has a runner")
	ErrRunnerTaskNotStartable       = errors.New("runner task is not available to start")
	ErrRunnerTaskNotCompletable     = errors.New("runner task is not available to complete")
	ErrRunnerTaskCompletionConflict = errors.New("runner task has a different terminal result")
	ErrTaskLogUploadNotFound        = errors.New("runner task log upload not found")
	ErrTaskLogChunkSequenceConflict = errors.New("runner task log chunk sequence is not next")
	ErrTaskLogUploadFinalizing      = errors.New("runner task log upload is finalizing")
	ErrTaskLogUploadClosed          = errors.New("runner task log upload is closed")
)

/*
 * RunnerTask is the durable execution unit assigned to one fleet runner.
 */
type RunnerTask struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	FleetID           uuid.UUID
	RunnerID          *uuid.UUID
	Backend           string
	State             string
	Result            datatypes.JSON
	ExitCode          *int32
	ErrorMessage      *string
	CancelRequestedAt *time.Time
	QueuedAt          time.Time
	ReservedAt        *time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time

	/*
	 * PayloadCiphertext protects task inputs, which can contain credentials and other secrets.
	 */
	PayloadCiphertext []byte

	/*
	 * CompletionHash makes runner completion idempotent across reconnects.
	 * The WebSocket request ID only correlates an in-flight acknowledgement,
	 * while this hash detects a retry with the same terminal task result.
	 */
	CompletionHash *string
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

func (t *RunnerTask) IsTerminal() bool {
	switch t.State {
	case RunnerTaskStateSucceeded,
		RunnerTaskStateFailed,
		RunnerTaskStateCanceled,
		RunnerTaskStateLost:
		return true
	default:
		return false
	}
}

/*
 * RequestCancel is idempotent.
 * A queued task can be canceled immediately because no runner has observed it.
 * Assigned tasks retain their state and expose a cancellation request to the connected runner.
 */
func (t *RunnerTask) RequestCancel(tx *gorm.DB, now time.Time) error {
	if t.IsTerminal() {
		return nil
	}

	if t.State == RunnerTaskStateQueued && t.RunnerID == nil {
		result := tx.Model(t).
			Where("state = ? AND runner_id IS NULL", RunnerTaskStateQueued).
			Updates(map[string]any{
				"state":       RunnerTaskStateCanceled,
				"finished_at": now,
				"updated_at":  now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRunnerTaskNotReservable
		}
		t.State = RunnerTaskStateCanceled
		t.FinishedAt = &now
		t.UpdatedAt = now
		return nil
	}

	if t.CancelRequestedAt != nil {
		return nil
	}
	result := tx.Model(t).
		Where("state IN ?", []string{
			RunnerTaskStateReserved,
			RunnerTaskStateRunning,
		}).
		Where("cancel_requested_at IS NULL").
		Updates(map[string]any{
			"cancel_requested_at": now,
			"updated_at":          now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRunnerTaskNotCompletable
	}
	t.CancelRequestedAt = &now
	t.UpdatedAt = now
	return nil
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
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&TaskLogUpload{
		TaskID:    t.ID,
		UpdatedAt: now,
	}).Error; err != nil {
		return err
	}

	t.RunnerID = &runnerID
	t.State = RunnerTaskStateReserved
	t.ReservedAt = &now
	t.UpdatedAt = now
	return nil
}

func (r *Runner) FindActiveTask(tx *gorm.DB) (*RunnerTask, error) {
	var task RunnerTask
	err := tx.
		Where("runner_id = ? AND state IN ?", r.ID, []string{
			RunnerTaskStateReserved,
			RunnerTaskStateRunning,
		}).
		Order("reserved_at ASC").
		First(&task).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *Runner) ReserveNextTask(tx *gorm.DB) (*RunnerTask, error) {
	var task RunnerTask
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("fleet_id = ? AND backend = ? AND state = ? AND runner_id IS NULL",
			r.FleetID,
			RunnerTaskBackendIntegrated,
			RunnerTaskStateQueued,
		).
		Order("queued_at ASC, id ASC").
		First(&task).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := task.Reserve(tx, r.ID); err != nil {
		return nil, err
	}
	return &task, nil
}

func (t *RunnerTask) Start(tx *gorm.DB, runner *Runner, now time.Time) error {
	if t.RunnerID == nil || *t.RunnerID != runner.ID || t.State != RunnerTaskStateReserved {
		return ErrRunnerTaskNotStartable
	}

	taskResult := tx.Model(t).
		Where("runner_id = ? AND state = ?", runner.ID, RunnerTaskStateReserved).
		Updates(map[string]any{
			"state":      RunnerTaskStateRunning,
			"started_at": now,
			"updated_at": now,
		})
	if taskResult.Error != nil {
		return taskResult.Error
	}
	if taskResult.RowsAffected != 1 {
		return ErrRunnerTaskNotStartable
	}

	runnerResult := tx.Model(runner).
		Where("state = ?", RunnerStateIdle).
		Updates(map[string]any{
			"state":      RunnerStateBusy,
			"updated_at": now,
		})
	if runnerResult.Error != nil {
		return runnerResult.Error
	}
	if runnerResult.RowsAffected != 1 {
		return ErrRunnerTaskNotStartable
	}

	t.State = RunnerTaskStateRunning
	t.StartedAt = &now
	t.UpdatedAt = now
	runner.State = RunnerStateBusy
	runner.UpdatedAt = now
	return nil
}

func (t *RunnerTask) Complete(
	tx *gorm.DB,
	runner *Runner,
	completionHash string,
	result datatypes.JSON,
	exitCode int32,
	errorMessage string,
	canceled bool,
	now time.Time,
) error {
	var current RunnerTask
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND runner_id = ?", t.ID, runner.ID).
		First(&current).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRunnerTaskNotCompletable
		}
		return err
	}
	*t = current

	if t.IsTerminal() {
		if t.CompletionHash != nil && *t.CompletionHash == completionHash {
			return nil
		}
		return ErrRunnerTaskCompletionConflict
	}
	if t.State != RunnerTaskStateRunning {
		return ErrRunnerTaskNotCompletable
	}

	state := RunnerTaskStateSucceeded
	if canceled {
		state = RunnerTaskStateCanceled
	} else if exitCode != 0 || errorMessage != "" {
		state = RunnerTaskStateFailed
	}

	var storedError *string
	if errorMessage != "" {
		storedError = &errorMessage
	}
	taskResult := tx.Model(t).Updates(map[string]any{
		"state":           state,
		"result":          result,
		"exit_code":       exitCode,
		"error_message":   storedError,
		"completion_hash": completionHash,
		"finished_at":     now,
		"updated_at":      now,
	})
	if taskResult.Error != nil {
		return taskResult.Error
	}

	runnerUpdates := map[string]any{
		"updated_at": now,
	}
	if runner.Ephemeral {
		runnerUpdates["state"] = RunnerStateTerminated
		runnerUpdates["termination_reason"] = RunnerTerminationTaskCompleted
		runnerUpdates["terminated_at"] = now
		runner.State = RunnerStateTerminated
		reason := RunnerTerminationTaskCompleted
		runner.TerminationReason = &reason
		runner.TerminatedAt = &now
	} else {
		runnerUpdates["state"] = RunnerStateIdle
		runnerUpdates["termination_reason"] = nil
		runnerUpdates["terminated_at"] = nil
		runner.State = RunnerStateIdle
	}
	runnerResult := tx.Model(runner).
		Where("state = ?", RunnerStateBusy).
		Updates(runnerUpdates)
	if runnerResult.Error != nil {
		return runnerResult.Error
	}
	if runnerResult.RowsAffected != 1 {
		return ErrRunnerTaskNotCompletable
	}
	t.State = state
	t.Result = result
	t.ExitCode = &exitCode
	t.ErrorMessage = storedError
	t.CompletionHash = &completionHash
	t.FinishedAt = &now
	t.UpdatedAt = now
	runner.UpdatedAt = now
	return nil
}

/*
 * TaskLogUpload tracks only an active task log upload.
 * It provides the next accepted chunk sequence and byte count while SuperPlane receives or compacts chunks.
 * SuperPlane removes this transient row after it publishes the final compressed log object.
 */
type TaskLogUpload struct {
	TaskID            uuid.UUID `gorm:"primaryKey"`
	NextChunkSequence int64
	TotalBytes        int64
	FinalizingAt      *time.Time
	UpdatedAt         time.Time

	/*
	 * ProcessingUntil is a short-lived worker claim.
	 * Workers acquire it with FOR UPDATE SKIP LOCKED before blob I/O and can reclaim it after expiry.
	 */
	ProcessingUntil *time.Time
}

func (TaskLogUpload) TableName() string {
	return "runner_task_log_uploads"
}

func FindTaskLogUpload(tx *gorm.DB, taskID uuid.UUID) (*TaskLogUpload, error) {
	var upload TaskLogUpload
	err := tx.Where("task_id = ?", taskID).First(&upload).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskLogUploadNotFound
	}
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

type TaskLogFinalization struct {
	TaskID            uuid.UUID
	OrganizationID    uuid.UUID
	NextChunkSequence int64
	FinalizingAt      *time.Time
	ProcessingUntil   time.Time
}

/*
 * ClaimTaskLogFinalization uses FOR UPDATE SKIP LOCKED only while it records a processing lease.
 * The transaction then closes before the worker starts blob I/O, so compaction does not hold a
 * database connection or row lock for the duration of a potentially slow upload.
 * Another worker can reclaim the row if the lease expires after a crash.
 */
func ClaimTaskLogFinalization(tx *gorm.DB, now, cleanupBefore, processingUntil time.Time) (*TaskLogFinalization, error) {
	var candidate TaskLogFinalization
	err := tx.Transaction(func(tx *gorm.DB) error {
		err := tx.Table("runner_task_log_uploads AS uploads").
			Select(
				"uploads.task_id, tasks.organization_id, uploads.next_chunk_sequence, uploads.finalizing_at",
			).
			Joins("JOIN runner_tasks AS tasks ON tasks.id = uploads.task_id").
			Clauses(clause.Locking{
				Strength: "UPDATE",
				Table:    clause.Table{Name: "uploads"},
				Options:  "SKIP LOCKED",
			}).
			Where("tasks.state IN ?", []string{
				RunnerTaskStateSucceeded,
				RunnerTaskStateFailed,
				RunnerTaskStateCanceled,
				RunnerTaskStateLost,
			}).
			Where("uploads.processing_until IS NULL OR uploads.processing_until <= ?", now).
			Where("uploads.finalizing_at IS NULL OR uploads.finalizing_at <= ?", cleanupBefore).
			Order("uploads.updated_at ASC").
			Take(&candidate).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		result := tx.Model(&TaskLogUpload{}).
			Where("task_id = ?", candidate.TaskID).
			Updates(map[string]any{
				"processing_until": processingUntil,
				"updated_at":       now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrTaskLogUploadNotFound
		}
		candidate.ProcessingUntil = processingUntil
		return nil
	})
	if err != nil {
		return nil, err
	}
	if candidate.TaskID == uuid.Nil {
		return nil, nil
	}
	return &candidate, nil
}

func MarkTaskLogFinalizing(tx *gorm.DB, taskID uuid.UUID, processingUntil, now time.Time) error {
	result := tx.Model(&TaskLogUpload{}).
		Where("task_id = ? AND processing_until = ? AND finalizing_at IS NULL", taskID, processingUntil).
		Updates(map[string]any{
			"finalizing_at":    now,
			"processing_until": nil,
			"updated_at":       now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrTaskLogUploadNotFound
	}
	return nil
}

func DeleteClaimedTaskLogUpload(tx *gorm.DB, taskID uuid.UUID, processingUntil time.Time) error {
	return tx.
		Where("task_id = ? AND processing_until = ?", taskID, processingUntil).
		Delete(&TaskLogUpload{}).
		Error
}

func DeleteTaskLogUpload(tx *gorm.DB, taskID uuid.UUID) error {
	return tx.Where("task_id = ?", taskID).Delete(&TaskLogUpload{}).Error
}

func ReleaseTaskLogFinalization(tx *gorm.DB, taskID uuid.UUID, processingUntil, now time.Time) error {
	return tx.Model(&TaskLogUpload{}).
		Where("task_id = ? AND processing_until = ?", taskID, processingUntil).
		Updates(map[string]any{
			"processing_until": nil,
			"updated_at":       now,
		}).
		Error
}

func (r *Runner) FindTaskLogUpload(tx *gorm.DB, taskID uuid.UUID) (*TaskLogUpload, *RunnerTask, error) {
	var task RunnerTask
	err := tx.Where("id = ? AND runner_id = ?", taskID, r.ID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrTaskLogUploadNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	var upload TaskLogUpload
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("task_id = ?", taskID).
		First(&upload).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrTaskLogUploadNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &upload, &task, nil
}

func (u *TaskLogUpload) Advance(tx *gorm.DB, sequence, size int64, now time.Time) error {
	if u.FinalizingAt != nil {
		return ErrTaskLogUploadFinalizing
	}
	if sequence < u.NextChunkSequence {
		return nil
	}
	if sequence > u.NextChunkSequence {
		return ErrTaskLogChunkSequenceConflict
	}

	result := tx.Model(u).
		Where("next_chunk_sequence = ? AND finalizing_at IS NULL", sequence).
		Updates(map[string]any{
			"next_chunk_sequence": sequence + 1,
			"total_bytes":         gorm.Expr("total_bytes + ?", size),
			"updated_at":          now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrTaskLogChunkSequenceConflict
	}
	u.NextChunkSequence++
	u.TotalBytes += size
	u.UpdatedAt = now
	return nil
}
