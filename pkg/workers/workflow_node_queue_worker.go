package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

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

	event, err := w.findEvent(tx, queueItem)
	if err != nil {
		return err
	}

	config, err := w.buildNodeConfig(tx, queueItem, node, event)
	if err != nil {
		return err
	}

	comp, err := w.findComponent(node)
	if err != nil {
		return err
	}

	ctx := components.ProcessQueueContext{
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: config,
		RootEventID:   queueItem.RootEventID.String(),
		EventID:       event.ID.String(),
		Input:         event.Data.Data(),
	}

	ctx.CreateExecution = func() error {
		now := time.Now()

		execution := models.WorkflowNodeExecution{
			WorkflowID:          queueItem.WorkflowID,
			NodeID:              node.NodeID,
			RootEventID:         queueItem.RootEventID,
			EventID:             event.ID,
			PreviousExecutionID: event.ExecutionID,
			State:               models.WorkflowNodeExecutionStatePending,
			Configuration:       datatypes.NewJSONType(asMap(config)),
			CreatedAt:           &now,
			UpdatedAt:           &now,
		}

		err := tx.Create(&execution).Error
		if err != nil {
			return err
		}

		messages.NewWorkflowExecutionCreatedMessage(execution.WorkflowID.String(), &execution).PublishWithDelay(1 * time.Second)
		return nil
	}

	ctx.DequeueItem = func() error {
		return queueItem.Delete(tx)
	}

	ctx.UpdateNodeState = func(state string) error {
		return node.UpdateState(tx, state)
	}

	ctx.UpdateNodeState = func(state string) error {
		return node.UpdateState(tx, state)
	}

	ctx.DefaultProcessing = func() error {
		var err error

		err = ctx.CreateExecution()
		if err != nil {
			return err
		}

		err = ctx.DequeueItem()
		if err != nil {
			return err
		}

		return ctx.UpdateNodeState(models.WorkflowNodeStateProcessing)
	}

	return comp.ProcessQueueItem(ctx)
}

func (w *WorkflowNodeQueueWorker) findEvent(tx *gorm.DB, queueItem *models.WorkflowNodeQueueItem) (*models.WorkflowEvent, error) {
	event, err := models.FindWorkflowEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find event %s: %w", queueItem.EventID, err)
	}

	return event, nil
}

func (w *WorkflowNodeQueueWorker) buildNodeConfig(tx *gorm.DB, queueItem *models.WorkflowNodeQueueItem, node *models.WorkflowNode, event *models.WorkflowEvent) (any, error) {
	config, err := contexts.NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(event.Data.Data()).
		Build(node.Configuration.Data())

	if err != nil {
		return nil, err
	}

	return config, nil
}

func (w *WorkflowNodeQueueWorker) findComponent(node *models.WorkflowNode) (components.Component, error) {
	ref := node.Ref.Data()

	comp, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return nil, fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	return comp, nil
}

func (w *WorkflowNodeQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowNodeQueueWorker] "+format, v...)
}
