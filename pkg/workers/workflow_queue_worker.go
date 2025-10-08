package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WorkflowQueueWorker struct{}

func NewWorkflowQueueWorker() *WorkflowQueueWorker {
	return &WorkflowQueueWorker{}
}

func (w *WorkflowQueueWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processQueue(); err != nil {
				w.log("Error processing workflow queue: %v", err)
			}
		}
	}
}

func (w *WorkflowQueueWorker) processQueue() error {
	queueItems, err := models.FindOldestQueueItems()
	if err != nil {
		return err
	}

	for _, item := range queueItems {
		if err := w.processQueueItem(&item); err != nil {
			w.log("Error processing queue entry for node %s: %v", item.NodeID, err)
		}
	}

	return nil
}

func (w *WorkflowQueueWorker) processQueueItem(entry *models.WorkflowQueueItem) error {
	_, err := models.FindLastNodeExecutionForNode(
		entry.WorkflowID,
		entry.NodeID,
		[]string{models.WorkflowNodeExecutionStatePending, models.WorkflowNodeExecutionStateWaiting, models.WorkflowNodeExecutionStateStarted},
	)

	//
	// A pending/waiting/started execution already exists for this node.
	// Do not process this queue entry yet.
	//
	if err == nil {
		return nil
	}

	w.log("Creating new execution for workflow=%s, node=%s", entry.WorkflowID, entry.NodeID)

	//
	// Create new execution for workflow/node,
	// removing the event from the queue.
	//
	event, err := models.FindWorkflowEvent(entry.EventID.String())
	if err != nil {
		return fmt.Errorf("failed to find workflow event: %w", err)
	}

	workflow, err := models.FindWorkflow(entry.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to find workflow: %w", err)
	}

	node, err := workflow.FindNode(entry.NodeID)
	if err != nil {
		return fmt.Errorf("failed to find node %s: %w", entry.NodeID, err)
	}

	now := time.Now()
	execution := models.WorkflowNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    entry.WorkflowID,
		NodeID:        entry.NodeID,
		State:         models.WorkflowNodeExecutionStatePending,
		Inputs:        event.Data,
		Configuration: datatypes.NewJSONType(node.Configuration),
		EventID:       event.ID,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&execution).Error
		if err != nil {
			return err
		}

		return tx.
			Where("workflow_id = ?", entry.WorkflowID).
			Where("node_id = ?", entry.NodeID).
			Where("event_id = ?", entry.EventID).
			Delete(&models.WorkflowQueueItem{}).
			Error
	})
}

func (w *WorkflowQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowQueueWorker] "+format, v...)
}
