package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type OrganizationCleanupWorker struct {
	semaphore    *semaphore.Weighted
	logger       *log.Entry
	canvasWorker *CanvasCleanupWorker
}

func NewOrganizationCleanupWorker(providers ...agents.Provider) *OrganizationCleanupWorker {
	return &OrganizationCleanupWorker{
		semaphore:    semaphore.NewWeighted(10),
		logger:       log.WithFields(log.Fields{"worker": "OrganizationCleanupWorker"}),
		canvasWorker: NewCanvasCleanupWorker(providers...),
	}
}

func (w *OrganizationCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case tickTime := <-ticker.C:
			organizations, err := models.ListDeletedOrganizations()
			if err != nil {
				w.logger.Errorf("Error finding deleted organizations: %v", err)
				continue
			}

			for _, organization := range organizations {
				if deletedResourceWithinGracePeriod(organization.DeletedAt.Time, tickTime) {
					continue
				}

				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(organization models.Organization) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessOrganization(organization); err != nil {
						w.logger.Errorf("Error processing organization %s: %v", organization.ID, err)
					}
				}(organization)
			}
		}
	}
}

func (w *OrganizationCleanupWorker) LockAndProcessOrganization(organization models.Organization) error {
	if deletedResourceWithinGracePeriod(organization.DeletedAt.Time, time.Now()) {
		return nil
	}

	var sessionsToClean []models.AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedOrganization, err := models.LockDeletedOrganization(tx, organization.ID)
		if err != nil {
			w.logger.Infof("Organization %s already being processed - skipping", organization.ID)
			return nil
		}

		sessions, err := w.processOrganization(tx, *lockedOrganization)
		if err != nil {
			return err
		}

		sessionsToClean = sessions
		return nil
	})
	if err != nil {
		return err
	}

	w.canvasWorker.cleanupProviderSessions(context.Background(), sessionsToClean)
	return nil
}

func (w *OrganizationCleanupWorker) processOrganization(tx *gorm.DB, organization models.Organization) ([]models.AgentSession, error) {
	if !organization.DeletedAt.Valid {
		return nil, nil
	}

	canvases, err := models.ListMaybeDeletedCanvasesByOrganizationInTransaction(tx, organization.ID)
	if err != nil {
		return nil, fmt.Errorf("list organization canvases: %w", err)
	}

	var sessionsToClean []models.AgentSession
	for _, canvas := range canvases {
		if !canvas.DeletedAt.Valid {
			if err := canvas.SoftDeleteInTransaction(tx); err != nil {
				return nil, fmt.Errorf("soft delete canvas %s: %w", canvas.ID, err)
			}

			canvasInDB, err := models.FindUnscopedCanvasInTransaction(tx, canvas.ID)
			if err != nil {
				return nil, fmt.Errorf("reload soft-deleted canvas %s: %w", canvas.ID, err)
			}

			canvas = *canvasInDB
		}

		sessions, err := w.canvasWorker.processCanvas(tx, canvas)
		if err != nil {
			return nil, fmt.Errorf("process canvas %s: %w", canvas.ID, err)
		}
		sessionsToClean = append(sessionsToClean, sessions...)
	}

	var remainingCanvases int64
	err = tx.Unscoped().Model(&models.Canvas{}).Where("organization_id = ?", organization.ID).Count(&remainingCanvases).Error
	if err != nil {
		return nil, fmt.Errorf("count remaining canvases: %w", err)
	}

	if remainingCanvases > 0 {
		return sessionsToClean, nil
	}

	integrations, err := models.ListMaybeDeletedIntegrationsByOrganizationInTransaction(tx, organization.ID)
	if err != nil {
		return nil, fmt.Errorf("list organization integrations: %w", err)
	}

	for _, integration := range integrations {
		if integration.DeletedAt.Valid {
			continue
		}

		webhooks, err := models.ListIntegrationWebhooks(tx, integration.ID)
		if err != nil {
			return nil, fmt.Errorf("list integration webhooks for %s: %w", integration.ID, err)
		}

		for _, webhook := range webhooks {
			if err := tx.Delete(&webhook).Error; err != nil {
				return nil, fmt.Errorf("soft delete webhook %s: %w", webhook.ID, err)
			}
		}

		if err := integration.SoftDeleteInTransaction(tx); err != nil {
			return nil, fmt.Errorf("soft delete integration %s: %w", integration.ID, err)
		}
	}

	var remainingIntegrations int64
	err = tx.Unscoped().Model(&models.Integration{}).Where("organization_id = ?", organization.ID).Count(&remainingIntegrations).Error
	if err != nil {
		return nil, fmt.Errorf("count remaining integrations: %w", err)
	}

	if remainingIntegrations > 0 {
		return sessionsToClean, nil
	}

	organizationSessions, err := models.ListAgentSessionsForOrganizationInTransaction(tx, organization.ID)
	if err != nil {
		return nil, fmt.Errorf("list organization agent sessions: %w", err)
	}
	sessionsToClean = append(sessionsToClean, organizationSessions...)

	if err := models.DeleteAgentSessionsForOrganizationInTransaction(tx, organization.ID); err != nil {
		return nil, fmt.Errorf("delete organization agent sessions: %w", err)
	}

	if err := models.DeleteMetadataForOrganization(tx, models.DomainTypeOrganization, organization.ID.String()); err != nil {
		return nil, fmt.Errorf("delete organization role metadata: %w", err)
	}

	if err := tx.Where("organization_id = ?", organization.ID).Delete(&models.Blueprint{}).Error; err != nil {
		return nil, fmt.Errorf("delete organization blueprints: %w", err)
	}

	if err := tx.Where("domain_type = ?", models.DomainTypeOrganization).Where("domain_id = ?", organization.ID).Delete(&models.Secret{}).Error; err != nil {
		return nil, fmt.Errorf("delete organization secrets: %w", err)
	}

	if err := tx.Unscoped().Where("organization_id = ?", organization.ID).Delete(&models.User{}).Error; err != nil {
		return nil, fmt.Errorf("delete organization users: %w", err)
	}

	if err := tx.Unscoped().Delete(&organization).Error; err != nil {
		return nil, fmt.Errorf("hard delete organization: %w", err)
	}

	w.logger.Infof("Successfully cleaned up organization %s", organization.ID)
	return sessionsToClean, nil
}
