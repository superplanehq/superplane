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

    "github.com/google/uuid"
    "github.com/superplanehq/superplane/pkg/database"
    "github.com/superplanehq/superplane/pkg/grpc/actions/messages"
    "github.com/superplanehq/superplane/pkg/models"
    "github.com/superplanehq/superplane/pkg/workers/contexts"
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

    // Special handling for merge component: wait for all parents for this root
    if node.Ref.Data().Component.Name == "merge" {
        return w.processMergeNode(tx, node, queueItem)
    }

    w.log("De-queueing item %s for node=%s workflow=%s", queueItem.ID, node.NodeID, node.WorkflowID)
    return w.createNodeExecution(tx, node, queueItem)
}

// processMergeNode aggregates inputs from all immediate parents (for the same root event)
// and creates a single execution only when each parent has contributed at least one event.
func (w *WorkflowNodeQueueWorker) processMergeNode(tx *gorm.DB, node *models.WorkflowNode, pivot *models.WorkflowNodeQueueItem) error {
    workflow, err := models.FindUnscopedWorkflowInTransaction(tx, node.WorkflowID)
    if err != nil {
        return err
    }

    // Determine required parents (incoming edges)
    parents := workflow.FindIncomingEdges(node.NodeID)
    if len(parents) == 0 {
        // No parents? Treat like pass-through: just create execution for single input
        w.log("Merge node %s has no parents; proceeding with single input", node.NodeID)
        return w.createNodeExecution(tx, node, pivot)
    }

    // Collect all queue items for this node and root
    items, err := models.ListNodeQueueItemsForRoot(tx, node.WorkflowID, node.NodeID, pivot.RootEventID)
    if err != nil {
        return err
    }

    if len(items) == 0 {
        return nil
    }

    // Load events for items
    eventIDs := make([]string, 0, len(items))
    for _, it := range items {
        eventIDs = append(eventIDs, it.EventID.String())
    }
    events, err := models.FindWorkflowEvents(eventIDs)
    if err != nil {
        return err
    }

    // Check coverage: at least one event from every parent (match by source node id)
    have := map[string]bool{}
    for _, ev := range events {
        have[ev.NodeID] = true
    }
    allPresent := true
    for _, p := range parents {
        if !have[p.SourceID] {
            allPresent = false
            break
        }
    }

    if !allPresent {
        // Not ready yet; leave items in queue and try again later
        w.log("Merge waiting for all parents: node=%s root=%s have=%d need=%d", node.NodeID, pivot.RootEventID, len(have), len(parents))
        return nil
    }

    // Build aggregated input: simple list of all inputs
    aggregatedAll := make([]any, 0, len(events))
    for _, ev := range events {
        data := ev.Data.Data()
        aggregatedAll = append(aggregatedAll, data)
    }

    // Create a synthetic event that carries aggregated input; mark as routed so router ignores it
    now := time.Now()
    var syntheticInput any = aggregatedAll
    synthetic := models.WorkflowEvent{
        WorkflowID: node.WorkflowID,
        NodeID:     node.NodeID,
        Channel:    "default",
        Data:       datatypes.NewJSONType(syntheticInput),
        State:      models.WorkflowEventStateRouted,
        CreatedAt:  &now,
    }
    if err := tx.Create(&synthetic).Error; err != nil {
        return err
    }

    // Choose a previous execution ID from any parent event (if present)
    var prevExecID *uuid.UUID
    for _, ev := range events {
        if ev.ExecutionID != nil {
            prevExecID = ev.ExecutionID
            break
        }
    }

    // Build configuration and create execution
    config, err := contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).
        WithRootEvent(&pivot.RootEventID).
        WithPreviousExecution(prevExecID).
        WithInput(synthetic.Data.Data()).
        Build(node.Configuration.Data())
    if err != nil {
        return err
    }

    execNow := time.Now()
    nodeExecution := models.WorkflowNodeExecution{
        WorkflowID:          node.WorkflowID,
        NodeID:              node.NodeID,
        RootEventID:         pivot.RootEventID,
        EventID:             synthetic.ID,
        PreviousExecutionID: prevExecID,
        State:               models.WorkflowNodeExecutionStatePending,
        Configuration:       datatypes.NewJSONType(config),
        CreatedAt:           &execNow,
        UpdatedAt:           &execNow,
    }

    if err := tx.Create(&nodeExecution).Error; err != nil {
        return err
    }

    // Delete all consumed items for this root
    for _, it := range items {
        if err := it.Delete(tx); err != nil {
            return err
        }
    }

    messages.NewWorkflowExecutionCreatedMessage(nodeExecution.WorkflowID.String(), &nodeExecution).PublishWithDelay(1 * time.Second)

    return node.UpdateState(tx, models.WorkflowNodeStateProcessing)
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

func (w *WorkflowNodeQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowNodeQueueWorker] "+format, v...)
}
