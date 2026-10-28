package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListPendingRunnerTaskOrganizations traverses organizations by ID so a full
// organization cannot keep later organizations out of a bounded worker scan.
func ListPendingRunnerTaskOrganizations(tx *gorm.DB, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := pendingRunnerTasks(tx).
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

func pendingRunnerTasks(tx *gorm.DB) *gorm.DB {
	return tx.Model(&RunnerTask{}).
		Joins("JOIN runner_fleets ON runner_fleets.id = runner_tasks.fleet_id").
		Where("runner_tasks.backend = ? AND runner_tasks.state = ?", RunnerTaskBackendIntegrated, RunnerTaskStatePending).
		Where("runner_tasks.runner_id IS NULL AND runner_tasks.cancel_requested_at IS NULL").
		Where("runner_fleets.enabled = ? AND runner_fleets.deleted_at IS NULL", true)
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
