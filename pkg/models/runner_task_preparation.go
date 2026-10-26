package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const RunnerPreparationBudget = 5 * time.Minute
const RunnerPreparationAttemptTimeout = 30 * time.Second

var ErrRunnerPreparationUnavailable = errors.New("runner preparation is not available")
var ErrRunnerPreparationExpired = errors.New("runner preparation deadline expired")

// RunnerTaskPreparation survives reconnects, reassignment, and gateway restarts.
// AttemptID fences both successful payload publication and failed attempts.
type RunnerTaskPreparation struct {
	TaskID             uuid.UUID `gorm:"primaryKey"`
	StartedAt          time.Time
	Deadline           time.Time
	AttemptID          *uuid.UUID
	AttemptExpiresAt   *time.Time
	AttemptCount       int
	NextRetryAt        time.Time
	PreparedAt         *time.Time
	PreparedValidUntil *time.Time
	UpdatedAt          time.Time
}

// RunnerCredentialCleanup is written before a provider key is minted. Name
// permits reconciliation when creation succeeds but its response is lost.
type RunnerCredentialCleanup struct {
	ID                       uuid.UUID `gorm:"primaryKey"`
	TaskID                   uuid.UUID
	AttemptID                uuid.UUID
	Provider                 string
	ProviderResourceID       uuid.UUID
	IssuerCredentialIdentity string
	Name                     string
	KeyHash                  string
	ExpiresAt                time.Time
	AcceptedAt               *time.Time
	CompletedAt              *time.Time
	NextRetryAt              time.Time
	AttemptCount             int
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func ClaimRunnerTaskPreparation(tx *gorm.DB, taskID, runnerID, connectionID uuid.UUID, now time.Time) (*RunnerTaskPreparation, error) {
	var claimed *RunnerTaskPreparation
	err := tx.Transaction(func(tx *gorm.DB) error {
		runner, err := FindRunner(tx.Clauses(clause.Locking{Strength: "UPDATE"}), runnerID)
		if err != nil {
			return err
		}
		if runner.CurrentConnectionID == nil || *runner.CurrentConnectionID != connectionID || runner.State == RunnerStateTerminated {
			return ErrRunnerPreparationUnavailable
		}
		task, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE"}), taskID)
		if err != nil {
			return err
		}
		if task.State != RunnerTaskStateReserved || task.RunnerID == nil || *task.RunnerID != runnerID || task.CancelRequestedAt != nil || task.DispatchStartedAt != nil {
			return ErrRunnerPreparationUnavailable
		}
		initial := RunnerTaskPreparation{TaskID: taskID, StartedAt: now, Deadline: now.Add(RunnerPreparationBudget), NextRetryAt: now, UpdatedAt: now}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&initial).Error; err != nil {
			return err
		}
		var preparation RunnerTaskPreparation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&preparation, "task_id = ?", taskID).Error; err != nil {
			return err
		}
		if preparation.NextRetryAt.After(now) {
			return ErrRunnerPreparationUnavailable
		}
		if preparation.PreparedAt != nil && preparation.PreparedValidUntil != nil && preparation.PreparedValidUntil.After(now) {
			claimed = &preparation
			return nil
		}
		if !preparation.Deadline.After(now) {
			return ErrRunnerPreparationExpired
		}
		if preparation.NextRetryAt.After(now) || preparation.AttemptExpiresAt != nil && preparation.AttemptExpiresAt.After(now) {
			return ErrRunnerPreparationUnavailable
		}
		attemptID := uuid.New()
		expiresAt := now.Add(RunnerPreparationAttemptTimeout)
		if expiresAt.After(preparation.Deadline) {
			expiresAt = preparation.Deadline
		}
		preparation.AttemptID = &attemptID
		preparation.PreparedAt = nil
		preparation.PreparedValidUntil = nil
		preparation.AttemptExpiresAt = &expiresAt
		preparation.AttemptCount++
		preparation.UpdatedAt = now
		if err := tx.Save(&preparation).Error; err != nil {
			return err
		}
		claimed = &preparation
		return nil
	})
	return claimed, err
}

func (p *RunnerTaskPreparation) Accept(tx *gorm.DB, runnerID, connectionID uuid.UUID, ciphertext []byte, validUntil, now time.Time, validate ...func(*gorm.DB) error) error {
	err := tx.Transaction(func(tx *gorm.DB) error {
		runner, err := FindRunner(tx.Clauses(clause.Locking{Strength: "UPDATE"}), runnerID)
		if err != nil {
			return err
		}
		if runner.CurrentConnectionID == nil || *runner.CurrentConnectionID != connectionID {
			return ErrRunnerPreparationUnavailable
		}
		task, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE"}), p.TaskID)
		if err != nil {
			return err
		}
		if task.State != RunnerTaskStateReserved || task.RunnerID == nil || *task.RunnerID != runnerID || task.CancelRequestedAt != nil || task.DispatchStartedAt != nil {
			return ErrRunnerPreparationUnavailable
		}
		if !validUntil.After(now) {
			return ErrRunnerPreparationUnavailable
		}
		for _, guard := range validate {
			if err := guard(tx); err != nil {
				return err
			}
		}
		result := tx.Model(p).Where("attempt_id = ? AND attempt_expires_at > ? AND deadline > ? AND prepared_at IS NULL", p.AttemptID, now, now).
			Updates(map[string]any{"prepared_at": now, "prepared_valid_until": validUntil, "attempt_expires_at": nil, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrRunnerPreparationUnavailable
		}
		if err := tx.Model(task).Update("payload_ciphertext", ciphertext).Error; err != nil {
			return err
		}
		return tx.Model(&RunnerCredentialCleanup{}).Where("task_id = ? AND attempt_id = ? AND completed_at IS NULL", p.TaskID, p.AttemptID).
			Update("accepted_at", now).Error
	})
	if err == nil {
		acceptedAt := now.Truncate(time.Microsecond)
		p.PreparedAt = &acceptedAt
		p.PreparedValidUntil = &validUntil
	}
	return err
}

func (p *RunnerTaskPreparation) Retry(tx *gorm.DB, retryAt, now time.Time) error {
	return tx.Model(p).Where("attempt_id = ? AND prepared_at IS NULL", p.AttemptID).
		Updates(map[string]any{"attempt_expires_at": nil, "next_retry_at": retryAt, "updated_at": now}).Error
}

func (p *RunnerTaskPreparation) DelayDelivery(tx *gorm.DB, retryAt, now time.Time) error {
	return tx.Model(p).Where("attempt_id = ? AND prepared_at = ?", p.AttemptID, p.PreparedAt).
		Updates(map[string]any{"next_retry_at": retryAt, "attempt_count": gorm.Expr("attempt_count + 1"), "updated_at": now}).Error
}

// BeginDispatch persists the certainly-unsent boundary before socket I/O.
func (t *RunnerTask) BeginDispatch(tx *gorm.DB, runnerID, connectionID uuid.UUID, expected *RunnerTaskPreparation, now time.Time, validate ...func(*gorm.DB) error) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		runner, err := FindRunner(tx.Clauses(clause.Locking{Strength: "UPDATE"}), runnerID)
		if err != nil {
			return err
		}
		if runner.CurrentConnectionID == nil || *runner.CurrentConnectionID != connectionID || runner.State == RunnerStateTerminated {
			return ErrRunnerPreparationUnavailable
		}
		if _, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE"}), t.ID); err != nil {
			return err
		}
		var preparation RunnerTaskPreparation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&preparation, "task_id = ?", t.ID).Error; err != nil {
			return err
		}
		if expected == nil || expected.PreparedAt == nil || preparation.PreparedAt == nil || !preparation.PreparedAt.Equal(*expected.PreparedAt) || expected.AttemptID == nil || preparation.AttemptID == nil || *expected.AttemptID != *preparation.AttemptID || preparation.PreparedValidUntil == nil || !preparation.PreparedValidUntil.After(now) {
			return ErrRunnerPreparationUnavailable
		}
		for _, guard := range validate {
			if err := guard(tx); err != nil {
				return err
			}
		}
		result := tx.Model(t).Where("runner_id = ? AND state = ? AND cancel_requested_at IS NULL AND dispatch_started_at IS NULL", runnerID, RunnerTaskStateReserved).
			Updates(map[string]any{"dispatch_started_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrRunnerPreparationUnavailable
		}
		t.DispatchStartedAt = &now
		return nil
	})
}

func (t *RunnerTask) FailPreparation(tx *gorm.DB, diagnostic string, expected *RunnerTaskPreparation, now time.Time) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		var runner *Runner
		if t.RunnerID != nil {
			var err error
			runner, err = FindRunner(tx.Clauses(clause.Locking{Strength: "UPDATE"}), *t.RunnerID)
			if err != nil {
				return err
			}
		}
		current, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE"}), t.ID)
		if err != nil {
			return err
		}
		if current.DispatchStartedAt != nil || current.StartedAt != nil || (current.State != RunnerTaskStateReserved && current.State != RunnerTaskStateQueued) {
			return ErrRunnerPreparationUnavailable
		}
		if current.RunnerID != nil && (runner == nil || *current.RunnerID != runner.ID) {
			return ErrRunnerPreparationUnavailable
		}
		var preparation RunnerTaskPreparation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&preparation, "task_id = ?", t.ID).Error; err != nil {
			return err
		}
		if expected != nil {
			if preparation.AttemptID == nil || expected.AttemptID == nil || *preparation.AttemptID != *expected.AttemptID || (preparation.PreparedAt == nil) != (expected.PreparedAt == nil) {
				return ErrRunnerPreparationUnavailable
			}
			if preparation.PreparedAt != nil && !preparation.PreparedAt.Equal(*expected.PreparedAt) {
				return ErrRunnerPreparationUnavailable
			}
			if preparation.PreparedAt == nil && (preparation.AttemptExpiresAt == nil || !preparation.AttemptExpiresAt.After(now)) {
				return ErrRunnerPreparationUnavailable
			}
		} else if preparation.Deadline.After(now) || preparation.PreparedAt != nil && preparation.PreparedValidUntil != nil && preparation.PreparedValidUntil.After(now) {
			return ErrRunnerPreparationUnavailable
		}
		if err := tx.Model(current).Updates(map[string]any{"state": RunnerTaskStateFailed, "error_message": diagnostic, "finished_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		*t = *current
		t.State, t.ErrorMessage, t.FinishedAt = RunnerTaskStateFailed, &diagnostic, &now
		if runner == nil {
			return nil
		}
		return tx.Model(runner).Where("state IN ?", []string{RunnerStateIdle, RunnerStateBusy}).Update("state", RunnerStateIdle).Error
	})
}

func ListExpiredRunnerPreparations(tx *gorm.DB, now time.Time) ([]RunnerTask, error) {
	var tasks []RunnerTask
	err := tx.Model(&RunnerTask{}).Select("runner_tasks.*").Joins("JOIN runner_task_preparations p ON p.task_id = runner_tasks.id").
		Where("runner_tasks.state IN ? AND runner_tasks.dispatch_started_at IS NULL AND p.deadline <= ? AND (p.prepared_at IS NULL OR p.prepared_valid_until <= ?)", []string{RunnerTaskStateReserved, RunnerTaskStateQueued}, now, now).
		Find(&tasks).Error
	return tasks, err
}

func ListRunnerCredentialCleanup(tx *gorm.DB, now time.Time) ([]RunnerCredentialCleanup, error) {
	var records []RunnerCredentialCleanup
	err := tx.Model(&RunnerCredentialCleanup{}).Select("runner_credential_cleanups.*").
		Joins("JOIN runner_tasks t ON t.id = runner_credential_cleanups.task_id").
		Joins("LEFT JOIN runner_task_preparations p ON p.task_id = t.id").
		Where("runner_credential_cleanups.completed_at IS NULL AND runner_credential_cleanups.next_retry_at <= ?", now).
		Where("t.state NOT IN ? OR (runner_credential_cleanups.accepted_at IS NULL AND (p.attempt_id IS DISTINCT FROM runner_credential_cleanups.attempt_id OR p.attempt_expires_at <= ?))", []string{RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning}, now).
		Order("runner_credential_cleanups.created_at").Limit(100).Find(&records).Error
	return records, err
}

func ListUnacceptedRunnerCredentials(tx *gorm.DB, taskID uuid.UUID) ([]RunnerCredentialCleanup, error) {
	var records []RunnerCredentialCleanup
	err := tx.Where("task_id = ? AND completed_at IS NULL", taskID).Find(&records).Error
	return records, err
}

func (c *RunnerCredentialCleanup) Create(tx *gorm.DB) error { return tx.Create(c).Error }

func (c *RunnerCredentialCleanup) RecordHash(tx *gorm.DB, hash string) error {
	c.KeyHash = hash
	return tx.Model(c).Update("key_hash", hash).Error
}

func (c *RunnerCredentialCleanup) Complete(tx *gorm.DB, now time.Time) error {
	return tx.Model(c).Updates(map[string]any{"completed_at": now, "updated_at": now}).Error
}

func (c *RunnerCredentialCleanup) Retry(tx *gorm.DB, now time.Time) error {
	return tx.Model(c).Updates(map[string]any{"next_retry_at": now.Add(time.Minute), "attempt_count": gorm.Expr("attempt_count + 1"), "updated_at": now}).Error
}

func (c *RunnerCredentialCleanup) Claim(tx *gorm.DB, now time.Time) (bool, error) {
	result := tx.Model(c).Where("completed_at IS NULL AND next_retry_at <= ?", now).
		Update("next_retry_at", now.Add(RunnerPreparationAttemptTimeout))
	return result.RowsAffected == 1, result.Error
}
