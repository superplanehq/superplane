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

	// If this is a merge component, apply join semantics: wait for all parents for the same root
	if isMergeNode(node) {
		ready, err := w.tryCreateMergeExecution(tx, node, queueItem)
		if err != nil {
			return err
		}

		// Not all parents have produced inputs yet; leave node ready and try later
		// TODO: Is this efficient?
		if !ready {
			return nil
		}

		return nil
	}

	w.log("De-queueing item %s for node=%s workflow=%s", queueItem.ID, node.NodeID, node.WorkflowID)
	return w.createNodeExecution(tx, node, queueItem)
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

func (w *WorkflowNodeQueueWorker) tryCreateMergeExecution(tx *gorm.DB, node *models.WorkflowNode, firstItem *models.WorkflowNodeQueueItem) (bool, error) {
	workflow, err := models.FindUnscopedWorkflowInTransaction(tx, node.WorkflowID)
	if err != nil {
		return false, fmt.Errorf("failed to find workflow: %w", err)
	}

	// Build set of required parent node IDs (incoming edges to this node)
	requiredParents := map[string]struct{}{}
	for _, edge := range workflow.Edges {
		if edge.TargetID == node.NodeID {
			requiredParents[edge.SourceID] = struct{}{}
		}
	}

	if len(requiredParents) == 0 {
		w.log("Merge node %s has no incoming edges; falling back to pass-through", node.NodeID)
		return false, nil
	}

	queueItems, err := w.fetchMergableQueueItems(tx, node, firstItem)

	if err != nil {
		return false, err
	}

	if len(queueItems) == 0 {
		return false, nil
	}

	// Load events for the gathered queue items to know their source (parent) node IDs and payloads
	eventIDs := make([]string, 0, len(queueItems))
	for _, qi := range queueItems {
		eventIDs = append(eventIDs, qi.EventID.String())
	}

	var events []models.WorkflowEvent
	if err := tx.Where("id IN ?", eventIDs).Find(&events).Error; err != nil {
		return false, fmt.Errorf("failed to load events for merge: %w", err)
	}

	eventsByID := make(map[string]models.WorkflowEvent, len(events))
	for _, e := range events {
		eventsByID[e.ID.String()] = e
	}

	selectedByParent := make(map[string]models.WorkflowNodeQueueItem)
	selectedEventByParent := make(map[string]models.WorkflowEvent)

	for _, qi := range queueItems {
		e, ok := eventsByID[qi.EventID.String()]
		if !ok {
			continue
		}

		parentID := e.NodeID
		if _, req := requiredParents[parentID]; !req {
			continue
		}

		if _, already := selectedByParent[parentID]; already {
			continue
		}

		selectedByParent[parentID] = qi
		selectedEventByParent[parentID] = e
	}

	// Determine if we can proceed even if not all parents produced inputs,
	// by excluding parents that, for this root event instance, will not route to this merge.
	coveredParents := make(map[string]struct{}, len(selectedByParent))
	for pid := range selectedByParent {
		coveredParents[pid] = struct{}{}
	}

	if len(coveredParents) != len(requiredParents) {
		// Pre-compute channels from each parent to this merge node
		parentAllowedChannels := make(map[string]map[string]struct{})
		for parentID := range requiredParents {
			chans := map[string]struct{}{}
			for _, edge := range workflow.Edges {
				if edge.SourceID == parentID && edge.TargetID == node.NodeID {
					chans[edge.Channel] = struct{}{}
				}
			}
			parentAllowedChannels[parentID] = chans
		}

		// For each missing parent, check if it has finished and emitted no events for channels to this merge
		for parentID := range requiredParents {
			if _, ok := coveredParents[parentID]; ok {
				continue
			}

			// Find latest execution for this parent and root event
			var parentExec models.WorkflowNodeExecution
			err := tx.
				Where("workflow_id = ?", node.WorkflowID).
				Where("node_id = ?", parentID).
				Where("root_event_id = ?", firstItem.RootEventID).
				Order("created_at DESC").
				Take(&parentExec).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// Parent hasn't executed yet
					return false, nil
				}
				return false, fmt.Errorf("failed to find parent execution for %s: %w", parentID, err)
			}

			// If not finished, keep waiting
			if parentExec.State != models.WorkflowNodeExecutionStateFinished {
				return false, nil
			}

			// If the parent execution failed, this merge can never be fulfilled for this root event.
			// Clean up any queued items for this merge/root and stop processing.
			if parentExec.Result == models.WorkflowNodeExecutionResultFailed {
				if err := tx.
					Where("workflow_id = ?", node.WorkflowID).
					Where("node_id = ?", node.NodeID).
					Where("root_event_id = ?", firstItem.RootEventID).
					Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
					return false, fmt.Errorf("failed to clean merge queue items after parent failure: %w", err)
				}
				w.log("Skipping merge for node=%s workflow=%s root_event=%s due to failed parent %s; cleaned pending items",
					node.NodeID, node.WorkflowID, firstItem.RootEventID, parentID)
				return false, nil
			}

			// Check outputs produced by this parent execution
			outputs, err := parentExec.GetOutputs()
			if err != nil {
				return false, fmt.Errorf("failed to get outputs for parent %s: %w", parentID, err)
			}

			// If none of the outputs are on channels that go to this merge, exclude this parent
			allowed := parentAllowedChannels[parentID]
			routedToMerge := false
			for _, ev := range outputs {
				if _, ok := allowed[ev.Channel]; ok {
					routedToMerge = true
					break
				}
			}

			if !routedToMerge {
				// Parent finished but did not route to this merge along any connected channel.
				// This join can never be fulfilled for this root event; clean pending items and stop.
				if err := tx.
					Where("workflow_id = ?", node.WorkflowID).
					Where("node_id = ?", node.NodeID).
					Where("root_event_id = ?", firstItem.RootEventID).
					Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
					return false, fmt.Errorf("failed to clean merge queue items after non-routing parent: %w", err)
				}
				w.log("Skipping merge for node=%s workflow=%s root_event=%s due to parent %s not routing to merge; cleaned pending items",
					node.NodeID, node.WorkflowID, firstItem.RootEventID, parentID)
				return false, nil
			}
		}
	}

	// If still not covered, keep waiting
	if len(coveredParents) != len(requiredParents) {
		return false, nil
	}

	// Aggregate inputs into a single map keyed by parent node ID
	aggregated := make(map[string]any, len(selectedByParent))
	for parentID, ev := range selectedEventByParent {
		aggregated[parentID] = ev.Data.Data()
	}

	// Create a synthetic input event for the merge execution, set as routed to avoid the router picking it up
	now := time.Now()
	aggEvent := models.WorkflowEvent{
		WorkflowID: node.WorkflowID,
		NodeID:     node.NodeID, // input for merge
		Channel:    "default",
		Data:       datatypes.NewJSONType(any(aggregated)),
		State:      models.WorkflowEventStateRouted,
		CreatedAt:  &now,
	}
	if err := tx.Create(&aggEvent).Error; err != nil {
		return false, fmt.Errorf("failed to create aggregate event for merge: %w", err)
	}

	// Build configuration with merged input
	config, err := contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).
		WithRootEvent(&firstItem.RootEventID).
		WithInput(aggregated).
		Build(node.Configuration.Data())
	if err != nil {
		return false, err
	}

	// Create the merge node execution
	nodeExecution := models.WorkflowNodeExecution{
		WorkflowID:    node.WorkflowID,
		NodeID:        node.NodeID,
		RootEventID:   firstItem.RootEventID,
		EventID:       aggEvent.ID,
		State:         models.WorkflowNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(config),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := tx.Create(&nodeExecution).Error; err != nil {
		return false, fmt.Errorf("failed to create merge node execution: %w", err)
	}

	// Delete the consumed queue items (one per parent)
	for _, qi := range selectedByParent {
		if err := qi.Delete(tx); err != nil {
			return false, fmt.Errorf("failed to delete consumed queue item %s: %w", qi.ID, err)
		}
	}

	// Notify and set node as processing
	messages.NewWorkflowExecutionCreatedMessage(nodeExecution.WorkflowID.String(), &nodeExecution).PublishWithDelay(1 * time.Second)
	if err := node.UpdateState(tx, models.WorkflowNodeStateProcessing); err != nil {
		return false, err
	}

	w.log("Created merge execution %s for node=%s workflow=%s with %d inputs", nodeExecution.ID, node.NodeID, node.WorkflowID, len(selectedByParent))
	return true, nil
}

func (w *WorkflowNodeQueueWorker) fetchMergableQueueItems(tx *gorm.DB, node *models.WorkflowNode, firstItem *models.WorkflowNodeQueueItem) ([]models.WorkflowNodeQueueItem, error) {
	var queueItems []models.WorkflowNodeQueueItem

	err := tx.
		Where("workflow_id = ?", node.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("root_event_id = ?", firstItem.RootEventID).
		Order("created_at ASC").
		Find(&queueItems).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list queue items for merge: %w", err)
	}

	return queueItems, nil
}

func (w *WorkflowNodeQueueWorker) log(format string, v ...any) {
	log.Printf("[WorkflowNodeQueueWorker] "+format, v...)
}
