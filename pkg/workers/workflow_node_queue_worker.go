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

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"

	"github.com/superplanehq/superplane/pkg/workers/mergenodequeueprocessor"
)

type WorkflowNodeQueueWorker struct {
	semaphore *semaphore.Weighted
}

func NewWorkflowNodeQueueWorker() *WorkflowNodeQueueWorker {
	return &WorkflowNodeQueueWorker{
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

	if isMergeNode(node) {
		w.log("Processing merge node %s for workflow=%s", node.NodeID, node.WorkflowID)
		_, err := mergenodequeueprocessor.Process(tx, node)
		if err != nil {
			return err
		}

		return nil
	} else {
		w.log("De-queueing item %s for node=%s workflow=%s", queueItem.ID, node.NodeID, node.WorkflowID)
		return w.createNodeExecution(tx, node, queueItem)
	}
}

func (w *WorkflowNodeQueueWorker) createNodeExecution(tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem) error {
	event, err := models.FindWorkflowEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return fmt.Errorf("failed to event %s: %w", queueItem.EventID, err)
	}

	config, err := contexts.NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(event.Data.Data()).
		Build(node.Configuration.Data())

	if err != nil {
		return err
	}

	now := time.Now()
	nodeExecution := models.WorkflowNodeExecution{
		WorkflowID:          queueItem.WorkflowID,
		NodeID:              node.NodeID,
		RootEventID:         queueItem.RootEventID,
		EventID:             event.ID,
		PreviousExecutionID: event.ExecutionID,
		State:               models.WorkflowNodeExecutionStatePending,
		Configuration:       datatypes.NewJSONType(config),
		CreatedAt:           &now,
		UpdatedAt:           &now,
	}

	err = tx.Create(&nodeExecution).Error
	if err != nil {
		return err
	}

	err = queueItem.Delete(tx)
	if err != nil {
		return err
	}

	messages.NewWorkflowExecutionCreatedMessage(nodeExecution.WorkflowID.String(), &nodeExecution).PublishWithDelay(1 * time.Second)

	return node.UpdateState(tx, models.WorkflowNodeStateProcessing)
}

// TODO: This is a stupid way to check if it's a merge node
func isMergeNode(node *models.WorkflowNode) bool {
	if node.Type != models.NodeTypeComponent {
		return false
	}

	ref := node.Ref.Data()

	// TODO: Especially this part, ultra stupid
	return ref.Component != nil && ref.Component.Name == "merge"
}

func (w *WorkflowNodeQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowNodeQueueWorker] "+format, v...)
}
