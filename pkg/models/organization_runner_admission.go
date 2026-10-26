package models

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrOrganizationRunnerCapacity = errors.New("organization parallel runner limit reached")

// ResolveOrganizationParallelRunnerLimit includes trial and subscription expiry.
// Billing disabled and a missing catalog limit both mean unlimited capacity.
func ResolveOrganizationParallelRunnerLimit(tx *gorm.DB, organizationID uuid.UUID) (*int, error) {
	if !BillingEnabled() {
		return nil, nil
	}
	plan, err := ResolveOrganizationBillingPlan(tx, organizationID)
	if err != nil {
		return nil, err
	}
	return plan.Catalog.ParallelTaskLimit, nil
}

func CountActiveOrganizationRunnerJobs(tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&RunnerTask{}).
		Where("organization_id = ? AND backend = ? AND state IN ?", organizationID,
			RunnerTaskBackendIntegrated, []string{RunnerTaskStateReserved, RunnerTaskStateRunning}).
		Count(&count).Error
	return count, err
}

func ListOrganizationRunnerFleetIDs(tx *gorm.DB, organizationID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Model(&RunnerTask{}).Where("organization_id = ? AND backend = ? AND state IN ?", organizationID, RunnerTaskBackendIntegrated,
		[]string{RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning}).
		Distinct().Pluck("fleet_id", &ids).Error
	return ids, err
}

// LockOrganizationRunnerAdmission must precede runner and job locks.
func LockOrganizationRunnerAdmission(tx *gorm.DB, organizationID uuid.UUID) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", organizationRunnerAdmissionKey(organizationID)).Error
}

func TryLockOrganizationRunnerAdmission(tx *gorm.DB, organizationID uuid.UUID) (bool, error) {
	var locked bool
	err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", organizationRunnerAdmissionKey(organizationID)).Scan(&locked).Error
	return locked, err
}

func checkOrganizationRunnerCapacity(tx *gorm.DB, organizationID uuid.UUID) error {
	limit, err := ResolveOrganizationParallelRunnerLimit(tx, organizationID)
	if err != nil || limit == nil {
		return err
	}
	count, err := CountActiveOrganizationRunnerJobs(tx, organizationID)
	if err != nil {
		return err
	}
	if count >= int64(*limit) {
		return ErrOrganizationRunnerCapacity
	}
	return nil
}

func organizationRunnerAdmissionKey(organizationID uuid.UUID) int64 {
	sum := sha256.Sum256(append([]byte("organization-runner-admission:"), organizationID[:]...))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
