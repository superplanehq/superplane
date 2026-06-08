package workers

import (
	"fmt"
	"log"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookBackfillJob is a one-shot admin job that creates WebhookSubscriptionBinding rows
// for existing workflow_nodes that have a webhook_id set but no active binding yet.
// Only nodes whose handler implements ScopeKeyer receive a binding.
// Safe to run multiple times — nodes that already have an active binding are skipped.
//
// Run via: RUN_WEBHOOK_BACKFILL_JOB=yes
type WebhookBackfillJob struct {
	registry *registry.Registry
}

func NewWebhookBackfillJob(r *registry.Registry) *WebhookBackfillJob {
	return &WebhookBackfillJob{registry: r}
}

func (j *WebhookBackfillJob) Run() error {
	db := database.Conn()

	var nodes []models.CanvasNode
	if err := db.
		Where("webhook_id IS NOT NULL AND app_installation_id IS NOT NULL").
		Find(&nodes).Error; err != nil {
		return fmt.Errorf("listing nodes with webhooks: %w", err)
	}

	j.log("Found %d node(s) with webhook_id set", len(nodes))

	created, skipped, failed := 0, 0, 0
	for _, node := range nodes {
		ok, err := j.backfillNode(db, node)
		if err != nil {
			j.log("Error backfilling node workflow=%s node=%s: %v", node.WorkflowID, node.NodeID, err)
			failed++
			continue
		}
		if ok {
			created++
		} else {
			skipped++
		}
	}

	j.log("Backfill complete: created=%d skipped=%d failed=%d", created, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d node(s) failed to backfill", failed)
	}
	return nil
}

// backfillNode returns (true, nil) when a binding was created, (false, nil) when skipped.
func (j *WebhookBackfillJob) backfillNode(db *gorm.DB, node models.CanvasNode) (bool, error) {
	var count int64
	if err := db.Model(&models.WebhookSubscriptionBinding{}).
		Where("workflow_id = ? AND node_id = ? AND active = true", node.WorkflowID, node.NodeID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking existing binding: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	webhook, err := models.FindWebhookInTransaction(db, *node.WebhookID)
	if err != nil {
		return false, fmt.Errorf("loading webhook %s: %w", *node.WebhookID, err)
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(db, *node.AppInstallationID)
	if err != nil {
		return false, fmt.Errorf("loading integration %s: %w", *node.AppInstallationID, err)
	}

	handler, err := j.registry.GetWebhookHandler(integration.AppName)
	if err != nil {
		return false, fmt.Errorf("getting webhook handler for %s: %w", integration.AppName, err)
	}

	sk, ok := handler.(core.ScopeKeyer)
	if !ok {
		return false, nil
	}

	config := webhook.Configuration.Data()
	scopeKey, err := sk.ScopeKey(config)
	if err != nil {
		return false, fmt.Errorf("deriving scope key: %w", err)
	}

	hash, err := opConfigHash(config)
	if err != nil {
		return false, fmt.Errorf("hashing config: %w", err)
	}

	now := time.Now()
	binding := models.WebhookSubscriptionBinding{
		OrganizationID:    integration.OrganizationID,
		AppInstallationID: integration.ID,
		WorkflowID:        node.WorkflowID,
		NodeID:            node.NodeID,
		WebhookID:         node.WebhookID,
		ScopeKey:          scopeKey,
		RequestedConfig:   datatypes.NewJSONType(config),
		RequestedHash:     hash,
		Active:            true,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	if err := db.Create(&binding).Error; err != nil {
		return false, fmt.Errorf("creating binding: %w", err)
	}

	return true, nil
}

func (j *WebhookBackfillJob) log(format string, v ...any) {
	log.Printf("[WebhookBackfillJob] "+format, v...)
}
