package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListRunnerTaskSchedulingOrganizations includes pending work and queued work
// that must release its slot because its fleet is unavailable. ID pagination
// prevents a full organization from hiding later work in a bounded scan.
func ListRunnerTaskSchedulingOrganizations(tx *gorm.DB, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := unassignedIntegratedRunnerTasks(tx).
		Where(`(runner_tasks.state = ? AND runner_tasks.cancel_requested_at IS NULL
			AND `+runnerTaskFleetAvailable+`)
			OR (runner_tasks.state = ? AND NOT (`+runnerTaskFleetAvailable+`))`,
			RunnerTaskStatePending, RunnerTaskStateQueued).
		Where("runner_tasks.organization_id > ?", after).
		Distinct("runner_tasks.organization_id").
		Order("runner_tasks.organization_id ASC").Limit(limit).
		Pluck("runner_tasks.organization_id", &ids).Error
	return ids, err
}

// ScheduleOrganizationRunnerTasks acquires organization slots before exposing
// tasks to runners or fleet scaling. The returned tasks are committed when
// called without an outer transaction.
func ScheduleOrganizationRunnerTasks(tx *gorm.DB, organizationID uuid.UUID, batchSize int) ([]RunnerTask, error) {
	var scheduled []RunnerTask
	err := tx.Transaction(func(tx *gorm.DB) error {
		if err := lockRunnerTaskOrganization(tx, organizationID, "SKIP LOCKED"); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := lockOrganizationTaskFleets(tx, organizationID); err != nil {
			return err
		}
		slots, err := availableOrganizationRunnerSlots(tx, organizationID)
		if err != nil {
			return err
		}
		available := batchSize
		if slots != nil {
			available = min(available, *slots)
		}
		if available <= 0 {
			return nil
		}
		// Do not skip a locked task: newer tasks must not overtake the oldest
		// pending task in the same organization.
		if err := pendingRunnerTasks(tx).
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "runner_tasks"}}).
			Where("runner_tasks.organization_id = ?", organizationID).
			Select("runner_tasks.*").Order("runner_tasks.queued_at ASC, runner_tasks.id ASC").
			Limit(available).Find(&scheduled).Error; err != nil {
			return err
		}
		if len(scheduled) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(scheduled))
		now := time.Now()
		for i := range scheduled {
			ids = append(ids, scheduled[i].ID)
			scheduled[i].State = RunnerTaskStateQueued
			scheduled[i].ReadyAt = &now
			scheduled[i].UpdatedAt = now
		}
		return tx.Model(&RunnerTask{}).Where("id IN ?", ids).
			Updates(map[string]any{"state": RunnerTaskStateQueued, "ready_at": now, "updated_at": now}).Error
	})
	if err != nil {
		return nil, err
	}
	return scheduled, nil
}

// RequeueUnavailableFleetRunnerTasks releases slots held by unassigned tasks
// on disabled or deleted fleets. Reserved and running tasks keep their slots.
func RequeueUnavailableFleetRunnerTasks(tx *gorm.DB, organizationID uuid.UUID, batchSize int) ([]RunnerTask, error) {
	var tasks []RunnerTask
	err := tx.Transaction(func(tx *gorm.DB) error {
		if err := lockRunnerTaskOrganization(tx, organizationID, "SKIP LOCKED"); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := lockOrganizationTaskFleets(tx, organizationID); err != nil {
			return err
		}
		if err := unassignedIntegratedRunnerTasks(tx).
			Where("runner_tasks.organization_id = ? AND runner_tasks.state = ?", organizationID, RunnerTaskStateQueued).
			Where("NOT (" + runnerTaskFleetAvailable + ")").
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "runner_tasks"}, Options: "SKIP LOCKED"}).
			Select("runner_tasks.*").Order("runner_tasks.queued_at ASC, runner_tasks.id ASC").
			Limit(batchSize).Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(tasks))
		now := time.Now()
		for i := range tasks {
			ids = append(ids, tasks[i].ID)
			tasks[i].State = RunnerTaskStatePending
			tasks[i].ReadyAt = nil
			tasks[i].UpdatedAt = now
		}
		return tx.Model(&RunnerTask{}).Where("id IN ?", ids).
			Updates(map[string]any{"state": RunnerTaskStatePending, "ready_at": nil, "updated_at": now}).Error
	})
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func pendingRunnerTasks(tx *gorm.DB) *gorm.DB {
	return unassignedIntegratedRunnerTasks(tx).
		Where("runner_tasks.state = ? AND runner_tasks.cancel_requested_at IS NULL", RunnerTaskStatePending).
		Where(runnerTaskFleetAvailable)
}

const runnerTaskFleetAvailable = `runner_fleets.enabled AND runner_fleets.deleted_at IS NULL
 AND (runner_fleets.scope_type = 'installation' OR runner_fleets.scope_id = runner_tasks.organization_id)
 AND ` + organizationFleetFeatureAvailable

func lockOrganizationTaskFleets(tx *gorm.DB, organizationID uuid.UUID) error {
	var fleets []RunnerFleet
	return tx.Unscoped().Clauses(clause.Locking{Strength: "SHARE"}).
		Where("id IN (?)", tx.Model(&RunnerTask{}).Select("fleet_id").
			Where("organization_id = ? AND runner_id IS NULL AND state IN ?", organizationID,
				[]string{RunnerTaskStatePending, RunnerTaskStateQueued})).
		Order("id ASC").Find(&fleets).Error
}

func unassignedIntegratedRunnerTasks(tx *gorm.DB) *gorm.DB {
	return tx.Model(&RunnerTask{}).
		Joins("JOIN runner_fleets ON runner_fleets.id = runner_tasks.fleet_id").
		Where("runner_tasks.backend = ? AND runner_tasks.runner_id IS NULL", RunnerTaskBackendIntegrated)
}

func lockRunnerTaskOrganization(tx *gorm.DB, organizationID uuid.UUID, options string) error {
	var organization Organization
	// NO KEY UPDATE serializes admissions without blocking foreign-key checks
	// when another transaction adds queued work for the organization.
	return tx.Select("id").Clauses(clause.Locking{Strength: "NO KEY UPDATE", Options: options}).
		Where("id = ?", organizationID).First(&organization).Error
}

// availableOrganizationRunnerSlots returns nil for unlimited organizations.
// Ready, reserved, and running tasks retain the same organization slot.
func availableOrganizationRunnerSlots(tx *gorm.DB, organizationID uuid.UUID) (*int, error) {
	if !billing.Enabled() {
		return nil, nil
	}
	limit, err := billing.ResolveParallelTaskLimit(tx, organizationID)
	if err != nil {
		return nil, err
	}
	if limit == nil {
		return nil, nil
	}
	var used int64
	err = tx.Model(&RunnerTask{}).
		Where("organization_id = ? AND backend = ? AND state IN ?", organizationID,
			RunnerTaskBackendIntegrated, []string{RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning}).
		Count(&used).Error
	if err != nil {
		return nil, err
	}
	available := *limit - int(used)
	return &available, nil
}
