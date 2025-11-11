package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WorkflowNodeQueueWorker struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
	logger    *log.Entry
}

func NewWorkflowNodeQueueWorker(registry *registry.Registry) *WorkflowNodeQueueWorker {
	return &WorkflowNodeQueueWorker{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "WorkflowNodeQueueWorker"}),
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
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, node := range nodes {
				logger := logging.ForNode(w.logger, node)
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(node models.WorkflowNode) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNode(logger, node); err != nil {
						logger.Errorf("Error processing: %v", err)
					}
				}(node)
			}
		}
	}
}

func (w *WorkflowNodeQueueWorker) LockAndProcessNode(logger *log.Entry, node models.WorkflowNode) error {
	var exec *models.WorkflowNodeExecution
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockWorkflowNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			logger.Info("Node already being processed - skipping")
			return nil
		}

		exec, err = w.processNode(tx, logger, n)
		return err
	})

	if err == nil && exec != nil {
		messages.NewWorkflowExecutionMessage(
			exec.WorkflowID.String(),
			exec.ID.String(),
			exec.NodeID,
		).Publish()
	}

	return err
}

func (w *WorkflowNodeQueueWorker) processNode(tx *gorm.DB, logger *log.Entry, node *models.WorkflowNode) (*models.WorkflowNodeExecution, error) {
	queueItem, err := node.FirstQueueItem(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	logger = logging.ForQueueItem(logger, *queueItem)
	logger.Info("Processing queue item")

	ctx, err := contexts.BuildProcessQueueContext(tx, node, queueItem)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

func (w *WorkflowNodeQueueWorker) processComponentNode(ctx *components.ProcessQueueContext, node *models.WorkflowNode) (*models.WorkflowNodeExecution, error) {
	ref := node.Ref.Data()

	if ref.Component == nil || ref.Component.Name == "" {
		return nil, fmt.Errorf("node %s has no component reference", node.NodeID)
	}

	comp, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return nil, fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	return comp.ProcessQueueItem(*ctx)
}
