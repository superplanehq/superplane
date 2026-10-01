package workers

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const intakeMonitorMetadataID = "monitorId"

// releaseRetiredDatadogMonitors deletes monitors for intakes that were
// removed by a committed canvas publish or canvas delete. It runs after
// commit so a failed publish cannot drop a monitor that the database
// restored.
func (w *WebhookCleanupWorker) releaseRetiredDatadogMonitors() {
	nodes, err := models.ListRetiredTriggerNodes(
		database.Conn(),
		(&datadog.OnErrorTrackingAlert{}).Name(),
		intakeMonitorMetadataID,
	)
	if err != nil {
		w.logger.Errorf("Error finding retired Datadog intakes: %v", err)
		return
	}

	for _, node := range nodes {
		if err := w.releaseRetiredDatadogMonitor(node); err != nil {
			w.logger.Errorf("Error releasing Datadog monitor for node %s on canvas %s: %v", node.NodeID, node.WorkflowID, err)
		}
	}
}

func (w *WebhookCleanupWorker) releaseRetiredDatadogMonitor(node models.CanvasNode) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		// Lock the canvas before the node. Canvas cleanup locks the canvas
		// first, and the reverse order can deadlock.
		canvas, err := models.LockUnscopedCanvas(tx, node.WorkflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		locked, err := models.LockUnscopedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		if !locked.DeletedAt.Valid && !canvas.DeletedAt.Valid {
			return nil
		}

		metadata := locked.Metadata.Data()
		if metadata == nil || metadata[intakeMonitorMetadataID] == nil || metadata[intakeMonitorMetadataID] == "" {
			return nil
		}
		if locked.AppInstallationID == nil {
			// The connection row is already gone. Integration cleanup deletes
			// owned monitors before that row is removed, and the foreign key
			// then clears this id. Drop the stored monitor id so the worker
			// does not retry.
			w.logger.Warnf("Datadog intake %s on canvas %s has no integration; clearing monitor id", locked.NodeID, locked.WorkflowID)
			return clearRetiredIntakeMonitor(tx, locked)
		}

		integration, err := models.FindMaybeDeletedIntegrationInTransaction(tx, *locked.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return clearRetiredIntakeMonitor(tx, locked)
			}
			return err
		}

		err = datadog.ReleaseRetiredIntakeMonitor(core.TriggerContext{
			Configuration: locked.Configuration.Data(),
			HTTP:          w.registry.HTTPContextInTransaction(tx),
			Integration: contexts.NewIntegrationContext(
				tx,
				locked,
				integration,
				w.encryptor,
				w.registry,
				nil,
			),
			Metadata: contexts.NewNodeMetadataContext(tx, locked),
			Logger:   logging.WithIntegration(logging.ForNode(*locked), *integration),
		})
		if err != nil {
			return err
		}

		return clearRetiredIntakeMonitor(tx, locked)
	})
}

func clearRetiredIntakeMonitor(tx *gorm.DB, node *models.CanvasNode) error {
	metadata := map[string]any{}
	for key, value := range node.Metadata.Data() {
		if key == intakeMonitorMetadataID || key == "monitorService" {
			continue
		}
		metadata[key] = value
	}

	return tx.Unscoped().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).
		Update("metadata", metadata).
		Error
}
