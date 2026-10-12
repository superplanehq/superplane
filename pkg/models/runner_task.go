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

	RunnerTaskLogStateActive     = "active"
	RunnerTaskLogStateArchivable = "archivable"
	RunnerTaskLogStateArchiving  = "archiving"
	RunnerTaskLogStateArchived   = "archived"
)

var (
	ErrRunnerTaskNotFound           = errors.New("runner task not found")
	ErrRunnerTaskNotReservable      = errors.New("runner task is not available for reservation")
	ErrRunnerTaskFleetMismatch      = errors.New("runner task fleet does not match runner fleet")
	ErrRunnerTaskBackendMismatch    = errors.New("runner task does not use the integrated backend")
	ErrRunnerTaskAlreadyAssigned    = errors.New("runner task already has a runner")
	ErrRunnerTaskNotStartable       = errors.New("runner task is not available to start")
	ErrRunnerTaskNotCompletable     = errors.New("runner task is not available to complete")
	ErrRunnerTaskLogStoreRequired   = errors.New("runner task active log store is required")
	ErrRunnerTaskCompletionConflict = errors.New("runner task has a different terminal result")
	ErrTaskLogLifecycleNotFound     = errors.New("runner task log lifecycle not found")
	ErrTaskLogChunkSequenceConflict = errors.New("runner task log chunk sequence is not next")
	ErrTaskLogLifecycleClosed       = errors.New("runner task log lifecycle is closed")
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

func (t *RunnerTask) Start(tx *gorm.DB, runner *Runner, activeLogStore string, now time.Time) error {
	if t.RunnerID == nil || *t.RunnerID != runner.ID || t.State != RunnerTaskStateReserved {
		return ErrRunnerTaskNotStartable
	}
	if activeLogStore == "" {
		return ErrRunnerTaskLogStoreRequired
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
	if err := tx.Create(&RunnerTaskLogLifecycle{
		TaskID:      t.ID,
		ActiveStore: activeLogStore,
		State:       RunnerTaskLogStateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error; err != nil {
		return err
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
) (bool, error) {
	var current RunnerTask
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND runner_id = ?", t.ID, runner.ID).
		First(&current).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrRunnerTaskNotCompletable
		}
		return false, err
	}
	*t = current

	if t.IsTerminal() {
		if t.CompletionHash != nil && *t.CompletionHash == completionHash {
			return false, nil
		}
		return false, ErrRunnerTaskCompletionConflict
	}
	if t.State != RunnerTaskStateRunning {
		return false, ErrRunnerTaskNotCompletable
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
		return false, taskResult.Error
	}
	if err := t.markLogsArchivable(tx, now); err != nil {
		return false, err
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
		return false, runnerResult.Error
	}
	if runnerResult.RowsAffected != 1 {
		return false, ErrRunnerTaskNotCompletable
	}
	t.State = state
	t.Result = result
	t.ExitCode = &exitCode
	t.ErrorMessage = storedError
	t.CompletionHash = &completionHash
	t.FinishedAt = &now
	t.UpdatedAt = now
	runner.UpdatedAt = now
	return true, nil
}

/*
 * RunnerTaskLogLifecycle coordinates an active store with final blob storage.
 * RunnerTask.Start creates this record when a reserved task starts running.
 *
 * Its state transitions are:
 *   - active: the runner can append chunks to the selected active store.
 *   - archivable: task completion moves the record to this state after the
 *     runner receives acknowledgements for all retained chunks. A running task
 *     that loses its runner also moves to this state.
 *   - archiving: the compactor claimed the record and is writing the final
 *     blob object.
 *   - archived: the final blob is available. CleanupAfter controls when the
 *     compactor deletes the now-redundant active-store data.
 *
 * After active-store cleanup, the record remains archived, FinalObjectKey
 * continues to identify the blob, and CleanupAfter is cleared.
 *
 * The selected active store owns append-frequency metadata such as the next
 * sequence, retained byte count, and authoritative truncation state. This
 * record can keep a low-frequency lifecycle summary, but the append path must
 * not update it for each chunk. When an external active store is selected,
 * accepting a new chunk must not cause a write to the application PostgreSQL
 * database.
 */
type RunnerTaskLogLifecycle struct {
	TaskID          uuid.UUID `gorm:"primaryKey"`
	ActiveStore     string
	State           string
	FinalObjectKey  *string
	FinalCursor     *string
	Truncated       bool
	CleanupAfter    *time.Time
	ProcessingUntil *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (t *RunnerTask) FindLifecycle(tx *gorm.DB) (*RunnerTaskLogLifecycle, error) {
	var lifecycle RunnerTaskLogLifecycle
	err := tx.Where("task_id = ?", t.ID).First(&lifecycle).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskLogLifecycleNotFound
	}
	if err != nil {
		return nil, err
	}
	return &lifecycle, nil
}

func (t *RunnerTask) markLogsArchivable(tx *gorm.DB, now time.Time) error {
	result := tx.Model(&RunnerTaskLogLifecycle{}).
		Where("task_id = ? AND state = ?", t.ID, RunnerTaskLogStateActive).
		Updates(map[string]any{
			"state":      RunnerTaskLogStateArchivable,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrTaskLogLifecycleNotFound
	}
	return nil
}
