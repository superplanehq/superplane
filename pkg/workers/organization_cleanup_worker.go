package workers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const DefaultOrganizationGracePeriodDays = 30

type OrganizationCleanupWorker struct {
	semaphore   *semaphore.Weighted
	logger      *log.Entry
	gracePeriod time.Duration
}

func NewOrganizationCleanupWorker() *OrganizationCleanupWorker {
	return &OrganizationCleanupWorker{
		semaphore:   semaphore.NewWeighted(5),
		logger:      log.WithFields(log.Fields{"worker": "OrganizationCleanupWorker"}),
		gracePeriod: organizationGracePeriod(),
	}
}

func organizationGracePeriod() time.Duration {
	days := DefaultOrganizationGracePeriodDays
	if v := os.Getenv("ORGANIZATION_CLEANUP_GRACE_PERIOD_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			days = parsed
		}
	}
	return time.Duration(days) * 24 * time.Hour
}

func (w *OrganizationCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			organizations, err := models.ListDeletedOrganizations()
			if err != nil {
				w.logger.Errorf("Error finding deleted organizations: %v", err)
				continue
			}

			for _, org := range organizations {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(org models.Organization) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessOrganization(org); err != nil {
						w.logger.Errorf("Error processing organization %s: %v", org.ID, err)
					}
				}(org)
			}
		}
	}
}

func (w *OrganizationCleanupWorker) LockAndProcessOrganization(org models.Organization) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedOrg, err := models.LockOrganization(tx, org.ID)
		if err != nil {
			w.logger.Infof("Organization %s already being processed - skipping", org.ID)
			return nil
		}

		w.logger.Infof("Processing deleted organization %s", lockedOrg.ID)
		return w.processOrganization(tx, *lockedOrg)
	})
}

func (w *OrganizationCleanupWorker) processOrganization(tx *gorm.DB, org models.Organization) error {
	if !org.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted organization %s", org.ID)
		return nil
	}

	if time.Since(org.DeletedAt.Time) < w.gracePeriod {
		w.logger.Infof("Organization %s is within grace period (deleted at %s, grace period %s) - skipping",
			org.ID, org.DeletedAt.Time.Format(time.RFC3339), w.gracePeriod)
		return nil
	}

	hasRemaining, err := w.hasRemainingChildResources(tx, org)
	if err != nil {
		return fmt.Errorf("failed to check remaining child resources: %w", err)
	}

	if hasRemaining {
		w.logger.Infof("Organization %s still has child resources being cleaned up - skipping hard delete", org.ID)
		return nil
	}

	if err := tx.Unscoped().Delete(&org).Error; err != nil {
		return fmt.Errorf("failed to hard-delete organization: %w", err)
	}

	w.logger.Infof("Successfully hard-deleted organization %s", org.ID)
	return nil
}

func (w *OrganizationCleanupWorker) hasRemainingChildResources(tx *gorm.DB, org models.Organization) (bool, error) {
	orgID := org.ID.String()

	var canvasCount int64
	if err := tx.Unscoped().Model(&models.Canvas{}).Where("organization_id = ?", orgID).Count(&canvasCount).Error; err != nil {
		return false, fmt.Errorf("failed to count canvases: %w", err)
	}
	if canvasCount > 0 {
		return true, nil
	}

	var integrationCount int64
	if err := tx.Unscoped().Model(&models.Integration{}).Where("organization_id = ?", orgID).Count(&integrationCount).Error; err != nil {
		return false, fmt.Errorf("failed to count integrations: %w", err)
	}
	if integrationCount > 0 {
		return true, nil
	}

	var userCount int64
	if err := tx.Unscoped().Model(&models.User{}).Where("organization_id = ?", orgID).Count(&userCount).Error; err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	if userCount > 0 {
		return true, nil
	}

	return false, nil
}
