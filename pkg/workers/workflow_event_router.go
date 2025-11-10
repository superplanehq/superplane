package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
)

type WorkflowEventRouter struct {
	semaphore *semaphore.Weighted
}

func NewWorkflowEventRouter() *WorkflowEventRouter {
	return &WorkflowEventRouter{
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *WorkflowEventRouter) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events, err := models.ListPendingWorkflowEvents()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, event := range events {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(event models.WorkflowEvent) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessEvent(event); err != nil {
						w.log("Error processing event %s: %v", event.ID, err)
					}
				}(event)
			}
		}
	}
}

func (w *WorkflowEventRouter) LockAndProcessEvent(event models.WorkflowEvent) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		e, err := models.LockWorkflowEvent(tx, event.ID)
		if err != nil {
			w.log("Execution already being processed - skipping")
			return nil
		}

		return w.processEvent(tx, e)
	})
}

func (w *WorkflowEventRouter) processEvent(tx *gorm.DB, event *models.WorkflowEvent) error {
	workflow, err := models.FindUnscopedWorkflowInTransaction(tx, event.WorkflowID)
	if err != nil {
		return err
	}

	if event.ExecutionID == nil {
		return w.processRootEvent(tx, workflow, event)
	}

	execution, err := models.FindNodeExecutionInTransaction(tx, event.WorkflowID, *event.ExecutionID)
	if err != nil {
		return err
	}

	if execution.ParentExecutionID != nil {
		return w.processChildExecutionEvent(tx, workflow, execution, event)
	}

	return w.processExecutionEvent(tx, workflow, execution, event)
}

func (w *WorkflowEventRouter) processRootEvent(tx *gorm.DB, workflow *models.Workflow, event *models.WorkflowEvent) error {
	now := time.Now()

	w.log("Processing root event %s", event.ID)

	edges := workflow.FindEdges(event.NodeID, event.Channel)
	for _, edge := range edges {
		targetNode, err := models.FindWorkflowNode(tx, workflow.ID, edge.TargetID)
		if err != nil {
			return err
		}

		queueItem := models.WorkflowNodeQueueItem{
			WorkflowID:  workflow.ID,
			NodeID:      targetNode.NodeID,
			RootEventID: event.ID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return err
		}

		// Notify via message bus (delayed to avoid race with transaction visibility)
		messages.NewWorkflowQueueItemCreatedMessage(workflow.ID.String(), &queueItem).PublishWithDelay(1 * time.Second)
	}

	return event.RoutedInTransaction(tx)
}

func (w *WorkflowEventRouter) processExecutionEvent(tx *gorm.DB, workflow *models.Workflow, execution *models.WorkflowNodeExecution, event *models.WorkflowEvent) error {
	now := time.Now()

	w.log("Processing event %s for execution %s", event.ID, execution.ID)

	edges := workflow.FindEdges(execution.NodeID, event.Channel)
	for _, edge := range edges {
		targetNode, err := models.FindWorkflowNode(tx, workflow.ID, edge.TargetID)
		if err != nil {
			return err
		}

		queueItem := models.WorkflowNodeQueueItem{
			WorkflowID:  workflow.ID,
			NodeID:      targetNode.NodeID,
			RootEventID: execution.RootEventID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return err
		}

		// Notify via message bus
		messages.NewWorkflowQueueItemCreatedMessage(workflow.ID.String(), &queueItem).PublishWithDelay(1 * time.Second)
	}

	return event.RoutedInTransaction(tx)
}

func (w *WorkflowEventRouter) processChildExecutionEvent(tx *gorm.DB, workflow *models.Workflow, execution *models.WorkflowNodeExecution, event *models.WorkflowEvent) error {
	w.log("Processing child execution event %s for execution %s", event.ID, execution.ID)

	parentExecution, err := models.FindNodeExecutionInTransaction(tx, workflow.ID, *execution.ParentExecutionID)
	if err != nil {
		return err
	}

	parentNode, err := models.FindWorkflowNode(tx, workflow.ID, parentExecution.NodeID)
	if err != nil {
		return err
	}

	blueprintID := parentNode.Ref.Data().Blueprint.ID
	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, blueprintID)
	if err != nil {
		return err
	}

	childNodeID := execution.NodeID[len(parentNode.NodeID)+1:]
	edges := blueprint.FindEdges(childNodeID, event.Channel)

	//
	// If there are no edges, it means the child node is a terminal node.
	// We should update the parent execution, if needed.
	//
	if len(edges) == 0 {

		//
		// Lock the parent execution to ensure we are not processing it multiple times for terminal nodes.
		//
		parentExecution, err := models.LockWorkflowNodeExecution(tx, *execution.ParentExecutionID)
		if err != nil {
			w.log("Child node %s is a terminal node, but parent is locked - skipping", childNodeID)
			return nil
		}

		w.log("Child node %s is a terminal node - checking parent execution", childNodeID)
		return w.completeParentExecutionIfNeeded(
			tx,
			parentNode,
			parentExecution,
			execution,
			event,
			blueprint,
		)
	}

	w.log("Child node %s is not a terminal node - creating next executions: %v", childNodeID, edges)

	//
	// Not a terminal node, create queue items for next internal nodes.
	// The queue worker will create child executions, preserving parent linkage.
	//
	now := time.Now()
	for _, edge := range edges {
		// Ensure target internal node exists as a workflow node
		targetNodeID := parentNode.NodeID + ":" + edge.TargetID
		if _, err := models.FindWorkflowNode(tx, workflow.ID, targetNodeID); err != nil {
			return err
		}

		queueItem := models.WorkflowNodeQueueItem{
			WorkflowID:  workflow.ID,
			NodeID:      targetNodeID,
			RootEventID: execution.RootEventID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return err
		}

		// Notify via message bus
		messages.NewWorkflowQueueItemCreatedMessage(workflow.ID.String(), &queueItem).PublishWithDelay(1 * time.Second)
	}

	return event.RoutedInTransaction(tx)
}

func (w *WorkflowEventRouter) completeParentExecutionIfNeeded(
	tx *gorm.DB,
	parentNode *models.WorkflowNode,
	parentExecution *models.WorkflowNodeExecution,
	execution *models.WorkflowNodeExecution,
	event *models.WorkflowEvent,
	blueprint *models.Blueprint,
) error {

	//
	// If the parent already finished, no need to do anything.
	//
	if parentExecution.State == models.WorkflowNodeExecutionStateFinished {
		w.log("Parent execution %s is already finished - skipping", parentExecution.ID)
		return event.RoutedInTransaction(tx)
	}

	//
	// Check if parent execution still has pending/started executions.
	//
	nonFinished, err := models.FindChildExecutionsInTransaction(tx, *execution.ParentExecutionID, []string{
		models.WorkflowNodeExecutionStatePending,
		models.WorkflowNodeExecutionStateStarted,
	})

	if err != nil {
		return err
	}

	//
	// If there are still pending/started executions, we should not complete the parent execution yet.
	//
	if len(nonFinished) > 0 {
		w.log("Parent execution %s still has %d pending/started executions - skipping", parentExecution.ID, len(nonFinished))
		return event.RoutedInTransaction(tx)
	}

	w.log("Parent execution %s has no more pending/started executions - completing", parentExecution.ID)

	finishedChildren, err := models.FindChildExecutionsInTransaction(tx, *execution.ParentExecutionID, []string{
		models.WorkflowNodeExecutionStateFinished,
	})

	if err != nil {
		return err
	}

	//
	// No more pending/started executions, we can complete the parent execution.
	//
	outputs := make(map[string][]any)
	for _, outputChannel := range blueprint.OutputChannels {
		fullNodeID := parentNode.NodeID + ":" + outputChannel.NodeID
		childExecutions := w.findChildrenForNode(finishedChildren, fullNodeID)
		if len(childExecutions) == 0 {
			continue
		}

		for _, childExecution := range childExecutions {
			outputEvents, err := childExecution.GetOutputsInTransaction(tx)
			if err != nil {
				return fmt.Errorf("error finding output events for %s: %v", fullNodeID, err)
			}

			for _, outputEvent := range outputEvents {
				if outputEvent.Channel == outputChannel.NodeOutputChannel {
					outputs[outputChannel.Name] = append(outputs[outputChannel.Name], outputEvent.Data.Data())
				}
			}
		}
	}

	events, err := parentExecution.PassInTransaction(tx, outputs)
	if err != nil {
		return err
	}

	messages.NewWorkflowExecutionFinishedMessage(parentExecution.WorkflowID.String(), parentExecution).PublishWithDelay(1 * time.Second)
	messages.PublishManyWorkflowEventsWithDelay(parentExecution.WorkflowID.String(), events, 1*time.Second)

	w.log("Parent execution %s completed", parentExecution.ID)
	return event.RoutedInTransaction(tx)
}

func (w *WorkflowEventRouter) findChildrenForNode(allChildren []models.WorkflowNodeExecution, nodeID string) []models.WorkflowNodeExecution {
	var childrenForNode []models.WorkflowNodeExecution
	for _, child := range allChildren {
		if child.NodeID == nodeID {
			childrenForNode = append(childrenForNode, child)
		}
	}

	return childrenForNode
}

func (w *WorkflowEventRouter) log(format string, v ...any) {
	log.Printf("[WorkflowEventRouter] "+format, v...)
}
