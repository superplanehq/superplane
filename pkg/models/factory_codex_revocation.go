package models

import (
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RevokeMemberCodexConnections runs inside member deletion, after the member
// row is locked. It never acquires execution or task locks.
func (u *User) RevokeCodexConnections(tx *gorm.DB, now time.Time) error {
	var connections []FactoryCodexConnection
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ? AND owner_user_id = ?", u.OrganizationID, u.ID).Order("id").Find(&connections).Error; err != nil {
		return err
	}
	for _, c := range connections {
		if err := tx.Model(&FactoryCodexConnection{}).Where("id = ?", c.ID).Updates(map[string]any{
			"status": CodexConnectionStatusDisconnected, "resume_status": "",
			"verification_url": "", "verification_code": "", "account_label": "",
			"usage_limits":     datatypes.NewJSONSlice([]CodexUsageLimit{}),
			"available_models": datatypes.NewJSONSlice([]CodexModel{}), "usage_observed_at": nil,
			"bundle_ciphertext": nil, "credential_version": 0, "final_checkpoint_version": 0,
			"reservation_generation": gorm.Expr("reservation_generation + 1"),
			"revocation_pending_at":  now, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&FactoryCodexLease{}).Where("connection_id = ?", c.ID).Updates(map[string]any{"revoked_at": now, "secret_hash": "", "secret_ciphertext": nil}).Error; err != nil {
			return err
		}
	}
	return nil
}

func ListPendingCodexRevocations(tx *gorm.DB) ([]FactoryCodexConnection, error) {
	var connections []FactoryCodexConnection
	err := tx.Where("revocation_pending_at IS NOT NULL").Order("revocation_pending_at, id").Limit(100).Find(&connections).Error
	return connections, err
}

// CleanupRevocation keeps durable evidence until every associated task and
// revoked runner has stopped. Each connection is processed in its own transaction.
func (c *FactoryCodexConnection) CleanupRevocation(tx *gorm.DB, now time.Time) ([]*RunCancellationResult, error) {
	var outcomes []*RunCancellationResult
	// Claim the connection before cleanup. Another worker can hold it, or can
	// finish the cleanup first. Then the saved snapshot is stale and must not
	// cancel runs or leases that started after a reconnect.
	var current FactoryCodexConnection
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ? AND revocation_pending_at IS NOT NULL", c.ID).First(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	*c = current
	var workspace Factory
	if err := tx.Unscoped().Where("id = ? AND organization_id = ?", c.FactoryID, c.OrganizationID).First(&workspace).Error; err != nil {
		return nil, err
	}
	var orders []FactoryWorkOrder
	err = tx.Where("organization_id = ? AND factory_id = ?", c.OrganizationID, c.FactoryID).
		Where("?", workspace.OnboardingConfigValue().CodexSubscription).Where("id IN (?)", tx.Model(&FactoryWorkOrderAssignee{}).Select("work_order_id").Where("user_id = ?", c.OwnerUserID)).Order("id").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	pending := false
	for i := range orders {
		o := &orders[i]
		// Skip tasks held by admission or completion without retaining locks on
		// their connection. The pending marker guarantees another attempt.
		var locked FactoryWorkOrder
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("id = ?", o.ID).First(&locked).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pending = true
			continue
		}
		if err != nil {
			return nil, err
		}
		// The task was reassigned between the query and the lock. Its new owner
		// may already have queued work or runs, so leave them alone.
		var owned int64
		if err := tx.Model(&FactoryWorkOrderAssignee{}).Where("work_order_id = ? AND user_id = ?", o.ID, c.OwnerUserID).Count(&owned).Error; err != nil {
			return nil, err
		}
		if owned == 0 {
			continue
		}
		if err := o.dropQueuedLineWork(tx); err != nil {
			return nil, err
		}
		runs, err := o.ListRuns(tx)
		if err != nil {
			return nil, err
		}
		for j := range runs {
			if runs[j].State == CanvasRunStateFinished || runs[j].State == CanvasRunStateCancelling {
				continue
			}
			result, err := runs[j].requestRunTermination(tx, nil, "")
			if err != nil {
				return nil, err
			}
			if result != nil {
				outcomes = append(outcomes, result)
			}
		}
		blocked, err := o.SubscriptionOwnershipBlocked(tx)
		if err != nil {
			return nil, err
		}
		pending = pending || blocked
	}
	var leases []FactoryCodexLease
	if err := tx.Where("connection_id = ?", c.ID).Order("id").Find(&leases).Error; err != nil {
		return nil, err
	}
	for _, lease := range leases {
		taskSnapshot, err := FindRunnerTask(tx, lease.TaskID)
		if errors.Is(err, ErrRunnerTaskNotFound) {
			if err := releaseConfirmedCodexLease(tx, lease.ID, now); err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		var runner *Runner
		if taskSnapshot.RunnerID != nil {
			runner, err = FindRunner(tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}), *taskSnapshot.RunnerID)
			if errors.Is(err, ErrRunnerNotFound) {
				if _, lookupErr := FindRunner(tx, *taskSnapshot.RunnerID); lookupErr == nil {
					pending = true
					continue
				} else if !errors.Is(lookupErr, ErrRunnerNotFound) {
					return nil, lookupErr
				}
				runner = nil
				err = nil
			}
			if err != nil {
				return nil, err
			}
		}
		task, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}), lease.TaskID)
		if errors.Is(err, ErrRunnerTaskNotFound) {
			// A skipped row and a missing row both retain evidence for the lease
			// cleanup, which checks existence again before it removes anything.
			pending = true
			continue
		}
		if err != nil {
			return nil, err
		}
		if err := task.RequestCancel(tx, now); err != nil {
			return nil, err
		}
		if runner != nil && task.IsTerminal() && runner.State != RunnerStateBusy {
			if err := runner.Terminate(tx, "codex_subscription_revoked"); err != nil {
				return nil, err
			}
		}
		if err := releaseConfirmedCodexLease(tx, lease.ID, now); err != nil {
			return nil, err
		}
	}
	count, err := CountCodexLeases(tx, c.ID)
	if err != nil {
		return nil, err
	}
	if !pending && count == 0 {
		err = tx.Model(&FactoryCodexConnection{}).Where("id = ? AND revocation_pending_at IS NOT NULL", c.ID).Update("revocation_pending_at", nil).Error
		if err == nil {
			c.RevocationPendingAt = nil
		}
	}
	return outcomes, err
}
