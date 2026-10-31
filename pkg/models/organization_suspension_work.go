package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const SuspensionWorkBatchSize = 100

func ListPendingOrganizationSuspensions(tx *gorm.DB, after uuid.UUID) ([]OrganizationSuspensionEvent, error) {
	var events []OrganizationSuspensionEvent
	query := tx.Where("action = ? AND cancellation_completed_at IS NULL", OrganizationSuspensionActionSuspend)
	if after != uuid.Nil {
		query = query.Where("id > ?", after)
	}
	err := query.Order("id").Limit(SuspensionWorkBatchSize).Find(&events).Error
	return events, err
}

// WithCurrentSuspension enforces the organization-first transition contract.
// A stale candidate never selects or mutates current work.
func (e *OrganizationSuspensionEvent) WithCurrentSuspension(db *gorm.DB, mutate func(*gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		o, err := LockOrganization(tx, e.OrganizationID)
		if err != nil {
			return err
		}
		var current OrganizationSuspensionEvent
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND action = ? AND cancellation_completed_at IS NULL", e.ID, e.OrganizationID, OrganizationSuspensionActionSuspend).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if o.SuspendedAt == nil || !o.SuspendedAt.Equal(current.CreatedAt) {
			return nil
		}
		return mutate(tx)
	})
}

func (e *OrganizationSuspensionEvent) ListRuns(tx *gorm.DB, after uuid.UUID) ([]CanvasRun, error) {
	var runs []CanvasRun
	query := tx.Where("workflow_id IN (?) AND state <> ?", suspensionCanvasIDs(tx, e.OrganizationID), CanvasRunStateFinished)
	if after != uuid.Nil {
		query = query.Where("id > ?", after)
	}
	err := query.Order("id").Limit(SuspensionWorkBatchSize).Find(&runs).Error
	return runs, err
}

func (e *OrganizationSuspensionEvent) CancelRunnerTasks(tx *gorm.DB, after uuid.UUID) ([]RunnerTask, error) {
	var tasks []RunnerTask
	query := tx.Where("organization_id = ? AND state IN ?", e.OrganizationID, []string{RunnerTaskStatePending, RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning})
	if after != uuid.Nil {
		query = query.Where("id > ?", after)
	}
	if err := query.Order("id").Limit(SuspensionWorkBatchSize).Find(&tasks).Error; err != nil {
		return nil, err
	}
	// Runner-before-task, after organization/event, matches reservation and completion.
	var runnerIDs []uuid.UUID
	for _, task := range tasks {
		if task.RunnerID != nil {
			runnerIDs = append(runnerIDs, *task.RunnerID)
		}
	}
	if len(runnerIDs) > 0 {
		var runners []Runner
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", runnerIDs).Order("id").Find(&runners).Error; err != nil {
			return nil, err
		}
	}
	for i := range tasks {
		current, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE"}), tasks[i].ID)
		if err != nil {
			return nil, err
		}
		if err := current.RequestCancel(tx, time.Now()); err != nil {
			return nil, err
		}
		tasks[i] = *current
	}
	return tasks, nil
}

func (e *OrganizationSuspensionEvent) DiscardQueuedWork(tx *gorm.DB) error {
	var items []FactoryWorkOrderQueueItem
	if err := tx.Where("organization_id = ?", e.OrganizationID).Order("line_id, id").Limit(SuspensionWorkBatchSize).Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		if _, err := lockFactoryLineForStepAdmission(tx, item.LineID); err != nil {
			return err
		}
		var current FactoryWorkOrderQueueItem
		err := tx.First(&current, "id = ?", item.ID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		dispatch, err := FindWorkOrderLineDispatch(tx, current.LineDispatchID)
		if err != nil {
			return err
		}
		if err := current.Delete(tx); err != nil {
			return err
		}
		if err := dispatch.Finish(tx, CanvasRunResultCancelled); err != nil {
			return err
		}
	}
	var requests []CanvasNodeRequest
	if err := e.pendingRequests(tx).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Order("id").Limit(SuspensionWorkBatchSize).Find(&requests).Error; err != nil {
		return err
	}
	for i := range requests {
		if err := requests[i].Complete(tx); err != nil {
			return err
		}
	}
	// Run-owned work drains under its run lock. Orphan queued work cannot replay.
	if _, err := deleteRowsLimited(tx, &CanvasNodeQueueItem{}, SuspensionWorkBatchSize, "workflow_id IN (?) AND run_id IS NULL", suspensionCanvasIDs(tx, e.OrganizationID)); err != nil {
		return err
	}
	ids := tx.Model(&CanvasEvent{}).Select("id").Where("workflow_id IN (?) AND run_id IS NULL AND state = ?", suspensionCanvasIDs(tx, e.OrganizationID), CanvasEventStatePending).Order("id").Limit(SuspensionWorkBatchSize)
	return tx.Model(&CanvasEvent{}).Where("id IN (?)", ids).Update("state", CanvasEventStateRouted).Error
}

func (e *OrganizationSuspensionEvent) PrepareAgentStops(tx *gorm.DB, after uuid.UUID) ([]AgentSession, error) {
	var sessions []AgentSession
	query := e.unconfirmedAgentStops(tx)
	if after != uuid.Nil {
		query = query.Where("id > ?", after)
	}
	if err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Limit(SuspensionWorkBatchSize).Find(&sessions).Error; err != nil {
		return nil, err
	}
	for i := range sessions {
		if sessions[i].Status == AgentSessionStatusStopping {
			continue
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		if err := tx.Model(&sessions[i]).Updates(map[string]any{"status": AgentSessionStatusStopping, "updated_at": now, "heartbeat_at": nil}).Error; err != nil {
			return nil, err
		}
		sessions[i].Status = AgentSessionStatusStopping
		sessions[i].UpdatedAt = &now
	}
	return sessions, nil
}

func (s *AgentSession) CompleteSuspensionStop(tx *gorm.DB, eventID uuid.UUID) error {
	return tx.Model(&AgentSession{}).Where("id = ? AND provider_session_id = ? AND status = ? AND updated_at = ?", s.ID, s.ProviderSessionID, AgentSessionStatusStopping, s.UpdatedAt).
		Updates(map[string]any{"status": AgentSessionStatusIdle, "updated_at": time.Now(), "last_active_at": time.Now(), "suspension_stop_event_id": eventID}).Error
}

func (s *AgentSession) CloseSuspensionTools(tx *gorm.DB) (bool, error) {
	ids := tx.Model(&AgentSessionMessage{}).Select("id").Where("session_id = ? AND role = ? AND tool_status = ?", s.ID, AgentMessageRoleTool, AgentToolStatusStarted).Order("id").Limit(SuspensionWorkBatchSize)
	if err := tx.Model(&AgentSessionMessage{}).Where("id IN (?)", ids).Update("tool_status", AgentToolStatusFinished).Error; err != nil {
		return false, err
	}
	var open int64
	err := tx.Model(&AgentSessionMessage{}).Where("session_id = ? AND role = ? AND tool_status = ?", s.ID, AgentMessageRoleTool, AgentToolStatusStarted).Count(&open).Error
	return open == 0, err
}

func (e *OrganizationSuspensionEvent) ListCancellingExecutions(tx *gorm.DB, after uuid.UUID) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	query := tx.Where("workflow_id IN (?) AND state = ?", suspensionCanvasIDs(tx, e.OrganizationID), CanvasNodeExecutionStateCancelling)
	if after != uuid.Nil {
		query = query.Where("id > ?", after)
	}
	err := query.Order("id").Limit(SuspensionWorkBatchSize).Find(&executions).Error
	return executions, err
}

func (e *OrganizationSuspensionEvent) CompleteCancellation(tx *gorm.DB) error {
	checks := []*gorm.DB{
		tx.Model(&FactoryWorkOrderQueueItem{}).Where("organization_id = ?", e.OrganizationID),
		tx.Model(&FactoryWorkOrderExecution{}).Where("organization_id = ? AND status IN ?", e.OrganizationID, []string{FactoryWorkOrderExecutionStatusPending, FactoryWorkOrderExecutionStatusRunning}),
		e.pendingRequests(tx),
		tx.Model(&CanvasRun{}).Where("workflow_id IN (?) AND state <> ?", suspensionCanvasIDs(tx, e.OrganizationID), CanvasRunStateFinished),
		tx.Model(&CanvasNodeExecution{}).Where("workflow_id IN (?) AND state IN ?", suspensionCanvasIDs(tx, e.OrganizationID), []string{CanvasNodeExecutionStatePending, CanvasNodeExecutionStateStarted, CanvasNodeExecutionStateCancelling}),
		tx.Model(&CanvasNodeQueueItem{}).Where("workflow_id IN (?)", suspensionCanvasIDs(tx, e.OrganizationID)),
		tx.Model(&CanvasEvent{}).Where("workflow_id IN (?) AND state = ?", suspensionCanvasIDs(tx, e.OrganizationID), CanvasEventStatePending),
		e.unconfirmedAgentStops(tx),
		tx.Model(&RunnerTask{}).Joins("LEFT JOIN runners r ON r.id = runner_tasks.runner_id").Where("runner_tasks.organization_id = ?", e.OrganizationID).
			Where("runner_tasks.state IN ? OR (runner_tasks.state = ? AND r.provider_deleted_at IS NULL AND r.id IS NOT NULL)", []string{RunnerTaskStatePending, RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning}, RunnerTaskStateLost),
	}
	for _, query := range checks {
		var exists bool
		if err := tx.Raw("SELECT EXISTS (?)", query.Select("1").Limit(1)).Scan(&exists).Error; err != nil {
			return err
		}
		if exists {
			return nil
		}
	}
	now := time.Now()
	if err := tx.Model(e).Update("cancellation_completed_at", now).Error; err != nil {
		return err
	}
	e.CancellationCompletedAt = &now
	return nil
}

func suspensionCanvasIDs(tx *gorm.DB, orgID uuid.UUID) *gorm.DB {
	return tx.Unscoped().Model(&Canvas{}).Select("id").Where("organization_id = ?", orgID)
}

func (e *OrganizationSuspensionEvent) unconfirmedAgentStops(tx *gorm.DB) *gorm.DB {
	// Local idle, failed, and terminated states do not prove remote work stopped.
	// Only this suspension's provider confirmation can release reactivation.
	return tx.Model(&AgentSession{}).Where("organization_id = ?", e.OrganizationID).
		Where("suspension_stop_event_id IS DISTINCT FROM ? OR status <> ?", e.ID, AgentSessionStatusIdle)
}

func (e *OrganizationSuspensionEvent) pendingRequests(tx *gorm.DB) *gorm.DB {
	return tx.Model(&CanvasNodeRequest{}).Where("workflow_id IN (?) AND state = ?", suspensionCanvasIDs(tx, e.OrganizationID), NodeExecutionRequestStatePending).
		Where(`NOT EXISTS (SELECT 1 FROM workflow_nodes n WHERE n.workflow_id = workflow_node_requests.workflow_id
		AND n.node_id = workflow_node_requests.node_id AND n.ref->'trigger'->>'name' = 'schedule'
		AND workflow_node_requests.execution_id IS NULL)`)
}
