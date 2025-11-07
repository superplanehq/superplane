package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WorkflowNodeQueueWorker struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
}

func NewWorkflowNodeQueueWorker(registry *registry.Registry) *WorkflowNodeQueueWorker {
	return &WorkflowNodeQueueWorker{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *WorkflowNodeQueueWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nodes, err := models.ListWorkflowNodesReady()
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

func (w *WorkflowNodeQueueWorker) LockAndProcessNode(node models.WorkflowNode) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockWorkflowNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			w.log("Node node=%s workflow=%s already being processed - skipping", node.NodeID, node.WorkflowID)
			return nil
		}

		return w.processNode(tx, n)
	})
}

func (w *WorkflowNodeQueueWorker) processNode(tx *gorm.DB, node *models.WorkflowNode) error {
	queueItem, err := node.FirstQueueItem(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	ctx, err := contexts.BuildProcessQueueContext(tx, node, queueItem)
	if err != nil {
		return err
	}

	switch node.Type {
	case models.NodeTypeComponent:
		/*
		* For component nodes, delegate to the component's ProcessQueueItem implementation to handle
		* the processing.
		 */
		return w.processComponentNode(ctx, node)
	case models.NodeTypeBlueprint:
		/*
		* For blueprint nodes, use the default processing logic.
		* Blueprint nodes do not have custom processing logic.
		 */
		return ctx.DefaultProcessing()
	default:
		return fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

func (w *WorkflowNodeQueueWorker) processComponentNode(ctx *components.ProcessQueueContext, node *models.WorkflowNode) error {
	ref := node.Ref.Data()

	if ref.Component == nil || ref.Component.Name == "" {
		return fmt.Errorf("node %s has no component reference", node.NodeID)
	}

	comp, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	return comp.ProcessQueueItem(*ctx)
}

func (w *WorkflowNodeQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowNodeQueueWorker] "+format, v...)
}
