package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

func (r *Runner) MarkLost(tx *gorm.DB, lastSeenBefore, now time.Time) error {
	if r.LastSeenAt == nil ||
		r.LastSeenAt.After(lastSeenBefore) ||
		(r.State != RunnerStateIdle && r.State != RunnerStateBusy) {
		return nil
	}

	task, taskErr := r.FindActiveTask(tx)
	switch {
	case taskErr == nil && task.State == RunnerTaskStateReserved:
		if err := tx.Model(task).Updates(map[string]any{
			"state":       RunnerTaskStateLost,
			"finished_at": now,
			"updated_at":  now,
		}).Error; err != nil {
			return err
		}
	case taskErr == nil && task.State == RunnerTaskStateRunning:
		if err := tx.Model(task).Updates(map[string]any{
			"state":       RunnerTaskStateLost,
			"finished_at": now,
			"updated_at":  now,
		}).Error; err != nil {
			return err
		}
		if err := task.markLogsArchivable(tx, now); err != nil {
			return err
		}
	case taskErr != nil && !errors.Is(taskErr, ErrRunnerTaskNotFound):
		return taskErr
	}

	reason := RunnerTerminationConnectionLost
	if err := tx.Model(r).Updates(map[string]any{
		"state":                    RunnerStateTerminated,
		"termination_reason":       reason,
		"terminated_at":            now,
		"updated_at":               now,
		"current_connection_id":    nil,
		"creation_idempotency_key": nil,
		"creation_request_hash":    nil,
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&RunnerCredential{}).
		Where("runner_id = ? AND revoked_at IS NULL", r.ID).
		Update("revoked_at", now).
		Error; err != nil {
		return err
	}

	r.State = RunnerStateTerminated
	r.TerminationReason = &reason
	r.TerminatedAt = &now
	r.UpdatedAt = now
	r.CurrentConnectionID = nil
	r.CreationIdempotencyKey = nil
	r.CreationRequestHash = nil
	return nil
}
