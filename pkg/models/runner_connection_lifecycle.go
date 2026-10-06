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

	previousState := r.State
	var idleStartedAt time.Time
	if previousState == RunnerStateIdle {
		idleStartedAt = r.idleStartedAt(tx)
	}

	task, taskErr := r.FindActiveTask(tx)
	var lostTask *RunnerTask
	var lostFromState string
	switch {
	case taskErr == nil && task.State == RunnerTaskStateReserved:
		lostTask = task
		lostFromState = task.State
		if err := tx.Model(task).Updates(map[string]any{
			"state":       RunnerTaskStateLost,
			"finished_at": now,
			"updated_at":  now,
		}).Error; err != nil {
			return err
		}
	case taskErr == nil && task.State == RunnerTaskStateRunning:
		lostTask = task
		lostFromState = task.State
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
	r.recordLostMetrics(tx, previousState, idleStartedAt, lostTask, lostFromState, now)
	return nil
}

func (r *Runner) recordLostMetrics(
	tx *gorm.DB,
	previousState string,
	idleStartedAt time.Time,
	task *RunnerTask,
	taskState string,
	now time.Time,
) {
	switch previousState {
	case RunnerStateIdle:
		if !idleStartedAt.IsZero() {
			recordRunnerStateOccupancy(tx, r.FleetID, RunnerStateIdle, now.Sub(idleStartedAt))
		}
	case RunnerStateBusy:
		if task != nil && task.StartedAt != nil {
			recordRunnerStateOccupancy(tx, r.FleetID, RunnerStateBusy, now.Sub(*task.StartedAt))
		}
	}

	if task == nil {
		return
	}
	switch taskState {
	case RunnerTaskStateReserved:
		recordRunnerTaskQueueWait(tx, task.FleetID, RunnerQueueWaitLost, task.QueuedAt, now)
	case RunnerTaskStateRunning:
		recordRunnerTaskRun(tx, task.FleetID, RunnerTaskStateLost, task.StartedAt, now)
	}
}
