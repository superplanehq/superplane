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
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WorkflowNodeQueueWorker struct {
    semaphore *semaphore.Weighted
    registry  *registry.Registry
}

func NewWorkflowNodeQueueWorker() *WorkflowNodeQueueWorker { return &WorkflowNodeQueueWorker{semaphore: semaphore.NewWeighted(25)} }

func NewWorkflowNodeQueueWorkerWithRegistry(reg *registry.Registry) *WorkflowNodeQueueWorker {
    return &WorkflowNodeQueueWorker{semaphore: semaphore.NewWeighted(25), registry: reg}
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

	w.log("De-queueing item %s for node=%s workflow=%s", queueItem.ID, node.NodeID, node.WorkflowID)
	return w.createNodeExecution(tx, node, queueItem)
}

func (w *WorkflowNodeQueueWorker) createNodeExecution(tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem) error {
    // Load component ref to see if it implements PreExecutionPolicy
    // If registry is not set or component not found or doesn't opt-in, fall back to legacy path.
    if w.registry != nil {
        ref := node.Ref.Data()
        if ref.Component != nil {
            if comp, err := w.registry.GetComponent(ref.Component.Name); err == nil {
                if policy, ok := comp.(components.PreExecutionPolicy); ok && policy.WantsPreExecution(node.Configuration.Data()) {
                    return w.createNodeExecutionWithPreExecutionPolicy(tx, node, queueItem, policy)
                }
            }
        }
    }

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

func (w *WorkflowNodeQueueWorker) createNodeExecutionWithPreExecutionPolicy(tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem, policy components.PreExecutionPolicy) error {
    // Gather all pending items for this node and root event
    // and evaluate readiness using the policy.

    // Find workflow to extract incoming edges
    workflow, err := models.FindUnscopedWorkflowInTransaction(tx, node.WorkflowID)
    if err != nil {
        return err
    }

    // Build incoming edges list for this node
    incoming := []components.IncomingEdge{}
    for _, e := range workflow.Edges {
        if e.TargetID == node.NodeID {
            incoming = append(incoming, components.IncomingEdge{SourceNodeID: e.SourceID, Channel: e.Channel})
        }
    }

    // Compute state key (root event based by default)
    _ = policy.StateKey(queueItem.RootEventID.String(), node.Configuration.Data())

    // List all queue items for this node to consider
    // We don’t have a state key column, so we filter by RootEventID which is part of the key.
    // Pull a reasonable batch; correctness does not depend on limit as we filter by root event.
    items, err := models.ListNodeQueueItems(node.WorkflowID, node.NodeID, 500, nil)
    if err != nil {
        return err
    }

    // Build observed map keyed by condition id and track consumed item IDs
    expected := policy.Expected(incoming, node.Configuration.Data())
    observed := map[string]any{}
    usedItems := map[string]models.WorkflowNodeQueueItem{}

    // Fetch events for items that share the same root event implied by the stateKey
    for _, it := range items {
        if it.RootEventID.String() != queueItem.RootEventID.String() {
            continue
        }
        evt, err := models.FindWorkflowEventInTransaction(tx, it.EventID)
        if err != nil {
            return err
        }
        // Map to a condition via source node and channel
        condID, obs := policy.Observe(evt.NodeID, evt.Channel, evt.Data.Data(), node.Configuration.Data())
        if condID == "" {
            continue
        }
        // Only keep first seen per condition id
        if _, ok := observed[condID]; !ok {
            observed[condID] = obs
            usedItems[condID] = it
        }
    }

    if !policy.Ready(expected, observed, node.Configuration.Data()) {
        // Not ready; leave items in queue
        w.log("Node %s waiting for join: %d/%d observed (root=%s)", node.NodeID, len(observed), len(expected), queueItem.RootEventID)
        return nil
    }

    // Aggregate payload
    agg := policy.Aggregate(expected, observed, node.Configuration.Data())

    now := time.Now()

    // Create execution with aggregated input
    config, err := contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).
        WithRootEvent(&queueItem.RootEventID).
        WithPreviousExecution(nil).
        WithInput(agg).
        Build(node.Configuration.Data())
    if err != nil {
        return err
    }

    nodeExecution := models.WorkflowNodeExecution{
        WorkflowID:    node.WorkflowID,
        NodeID:        node.NodeID,
        RootEventID:   queueItem.RootEventID,
        EventID:       queueItem.EventID,
        State:         models.WorkflowNodeExecutionStatePending,
        Configuration: datatypes.NewJSONType(config),
        CreatedAt:     &now,
        UpdatedAt:     &now,
    }

    if err := tx.Create(&nodeExecution).Error; err != nil {
        return err
    }

    // Consume used queue items
    for _, it := range usedItems {
        if err := it.Delete(tx); err != nil {
            return err
        }
    }

    messages.NewWorkflowExecutionCreatedMessage(nodeExecution.WorkflowID.String(), &nodeExecution).PublishWithDelay(1 * time.Second)
    return node.UpdateState(tx, models.WorkflowNodeStateProcessing)
}

func (w *WorkflowNodeQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowNodeQueueWorker] "+format, v...)
}
