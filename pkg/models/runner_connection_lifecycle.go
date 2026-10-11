package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// MarkLost returns the running task it ended, if any.
func (r *Runner) MarkLost(tx *gorm.DB, lastSeenBefore, now time.Time) (*RunnerTask, error) {
	if r.LastSeenAt == nil ||
		r.LastSeenAt.After(lastSeenBefore) ||
		(r.State != RunnerStateIdle && r.State != RunnerStateBusy) {
		return nil, nil
	}

	task, taskErr := r.FindActiveTask(tx)
	if taskErr != nil && !errors.Is(taskErr, ErrRunnerTaskNotFound) {
		return nil, taskErr
	}

	var lostRunningTask *RunnerTask
	if taskErr == nil {
		wasRunning := task.State == RunnerTaskStateRunning
		if err := tx.Model(task).Updates(map[string]any{
			"state":       RunnerTaskStateLost,
			"finished_at": now,
			"updated_at":  now,
		}).Error; err != nil {
			return nil, err
		}
		if wasRunning {
			if err := task.markLogsArchivable(tx, now); err != nil {
				return nil, err
			}
			lostRunningTask = task
		}
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
		return nil, err
	}
	if err := tx.Model(&RunnerCredential{}).
		Where("runner_id = ? AND revoked_at IS NULL", r.ID).
		Update("revoked_at", now).
		Error; err != nil {
		return nil, err
	}

	r.State = RunnerStateTerminated
	r.TerminationReason = &reason
	r.TerminatedAt = &now
	r.UpdatedAt = now
	r.CurrentConnectionID = nil
	r.CreationIdempotencyKey = nil
	r.CreationRequestHash = nil
	if lostRunningTask != nil {
		lostRunningTask.State = RunnerTaskStateLost
		lostRunningTask.FinishedAt = &now
		lostRunningTask.UpdatedAt = now
	}
	return lostRunningTask, nil
}
