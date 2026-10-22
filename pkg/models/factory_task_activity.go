package models

import (
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

// FactoryTaskActivityCounts is how many tasks were created, started, or
// finished in a window. Started means a step execution was created. A
// queued task has no execution yet, so it does not count. Finished means
// closed as completed. A task with more than one start in the window
// counts once.
type FactoryTaskActivityCounts struct {
	Created  int64
	Started  int64
	Finished int64
}

func CountFactoryTaskActivitySince(tx *gorm.DB, since time.Time) (FactoryTaskActivityCounts, error) {
	var counts FactoryTaskActivityCounts

	if err := tx.Model(&FactoryWorkOrder{}).Where("created_at >= ?", since).Count(&counts.Created).Error; err != nil {
		return counts, err
	}

	if err := tx.Raw(`
		SELECT COUNT(DISTINCT work_order_id)
		FROM factory_work_order_executions
		WHERE created_at >= ?
	`, since).Scan(&counts.Started).Error; err != nil {
		return counts, err
	}

	if err := tx.Raw(`
		SELECT COUNT(DISTINCT work_order_id)
		FROM factory_work_order_events
		WHERE type = ?
		  AND created_at >= ?
		  AND data->>'toState' = ?
		  AND data->>'toResult' = ?
	`, factory.EventTypeOrderStatusUpdated, since, FactoryWorkOrderStateClosed, FactoryWorkOrderResultCompleted).Scan(&counts.Finished).Error; err != nil {
		return counts, err
	}

	return counts, nil
}

// FirstInstallationAdminEmail returns the earliest installation admin email.
// An empty string means no admin exists yet.
func FirstInstallationAdminEmail(tx *gorm.DB) (string, error) {
	var account Account
	err := tx.Where("installation_admin = ?", true).Order("created_at ASC").First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return account.Email, nil
}

// FirstOrganizationName returns the earliest organization name.
// An empty string means no organization exists yet.
func FirstOrganizationName(tx *gorm.DB) (string, error) {
	var organization Organization
	err := tx.Order("created_at ASC").First(&organization).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return organization.Name, nil
}
