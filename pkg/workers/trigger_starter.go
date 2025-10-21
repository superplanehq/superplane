package workers

import (
	"context"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type TriggerStarter struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
}

func NewTriggerStarter(registry *registry.Registry, encryptor crypto.Encryptor) *TriggerStarter {
	return &TriggerStarter{
		registry:  registry,
		encryptor: encryptor,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *TriggerStarter) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nodes, err := models.ListReadyTriggers()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, node := range nodes {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(node models.WorkflowNode) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNode(node); err != nil {
						w.log("Error processing workflow node - workflow=%s, node=%s: %v", node.WorkflowID, node.NodeID, err)
					}
				}(node)
			}
		}
	}
}

func (w *TriggerStarter) LockAndProcessNode(node models.WorkflowNode) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockWorkflowNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			w.log("Node node=%s workflow=%s already being processed - skipping", node.NodeID, node.WorkflowID)
			return nil
		}

		return w.processNode(tx, n)
	})
}

func (w *TriggerStarter) processNode(tx *gorm.DB, node *models.WorkflowNode) error {
	ref := node.Ref.Data()
	if ref.Trigger == nil {
		w.log("Node %s is missing the trigger ref - skipping", node.NodeID)
		return nil
	}

	trigger, err := w.registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		w.log("Error getting trigger %s: %v", ref.Trigger.Name, err)
		return err
	}

	err = trigger.Start(triggers.TriggerContext{
		Configuration:   node.Configuration.Data(),
		EventContext:    contexts.NewEventContext(tx, node),
		MetadataContext: contexts.NewNodeMetadataContext(node),
		RequestContext:  contexts.NewNodeRequestContext(tx, node),
		WebhookContext:  contexts.NewWebhookContext(tx, context.Background(), w.encryptor, node),
	})

	if err != nil {
		w.log("Error starting trigger %s: %v", ref.Trigger.Name, err)
		return err
	}

	now := time.Now()
	node.State = models.WorkflowNodeStateProcessing
	node.UpdatedAt = &now
	return tx.Save(node).Error
}

func (w *TriggerStarter) log(format string, v ...any) {
	log.Printf("[TriggerStarter] "+format, v...)
}
