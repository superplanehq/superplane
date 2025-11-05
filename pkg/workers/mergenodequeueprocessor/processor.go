//
// MergeNodeQueueProcessor handles processing of merge nodes in the workflow.
// Merge nodes are special because they don't process individual queue items
// one by one, but rather wait for inputs from all required parent nodes
// before creating a single execution that aggregates those inputs.
//

package mergenodequeueprocessor

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/workers/workflow_node_queue_worker/contexts"
	"github.com/superplanehq/superplane/pkg/workers/workflow_node_queue_worker/models"
)

func Process(tx *gorm.DB, node *models.WorkflowNode) (bool, error) {
	p := &MergeNodeProcessor{tx: tx, node: node}

	p.findFirstQueueItem()
	p.loadWorkflow()
	p.buildRequiredParents()
	p.fetchMergableQueueItems()
	p.loadEvents()
	p.selectItemsByParent()
	p.validateMissingParents()
	p.aggregateInputs()
	p.createExecution()

	if p.err != nil {
		return false, p.err
	}

	return true, nil
}

type MergeNodeProcessor struct {
	tx   *gorm.DB
	node *models.WorkflowNode
	err  error

	firstItem             *models.WorkflowNodeQueueItem
	workflow              *models.Workflow
	requiredParents       map[string]struct{}
	queueItems            []models.WorkflowNodeQueueItem
	events                []models.WorkflowEvent
	eventsByID            map[string]models.WorkflowEvent
	selectedByParent      map[string]models.WorkflowNodeQueueItem
	selectedEventByParent map[string]models.WorkflowEvent
	coveredParents        map[string]struct{}
	aggregated            map[string]any
}

func (p *MergeNodeProcessor) findFirstQueueItem() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	var queueItem models.WorkflowNodeQueueItem
	err := p.tx.
		Where("workflow_id = ?", p.workflow.ID).
		Where("node_id = ?", p.node.ID).
		Order("created_at ASC").
		First(&queueItem).
		Error

	if err != nil {
		p.err = fmt.Errorf("failed to find first queue item for merge: %w", err)
	}

	return &queueItem, nil
}

func (p *MergeNodeProcessor) loadWorkflow() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	workflow, err := models.FindUnscopedWorkflowInTransaction(p.tx, p.node.WorkflowID)
	if err != nil {
		p.err = fmt.Errorf("failed to find workflow: %w", err)
		return p
	}

	p.workflow = workflow
	return p
}

func (p *MergeNodeProcessor) buildRequiredParents() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	p.requiredParents = map[string]struct{}{}
	for _, edge := range p.workflow.Edges {
		if edge.TargetID == p.node.NodeID {
			p.requiredParents[edge.SourceID] = struct{}{}
		}
	}

	if len(p.requiredParents) == 0 {
		p.w.log("Merge node %s has no incoming edges; falling back to pass-through", p.node.NodeID)
		p.err = fmt.Errorf("no required parents")
		return p
	}

	return p
}

func (p *MergeNodeProcessor) fetchMergableQueueItems() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	err := p.tx.
		Where("workflow_id = ?", p.node.WorkflowID).
		Where("node_id = ?", p.node.NodeID).
		Where("root_event_id = ?", p.firstItem.RootEventID).
		Order("created_at ASC").
		Find(&p.queueItems).Error

	if err != nil {
		p.err = fmt.Errorf("failed to list queue items for merge: %w", err)
		return p
	}

	if len(p.queueItems) == 0 {
		p.err = fmt.Errorf("no queue items")
		return p
	}

	return p
}

func (p *MergeNodeProcessor) loadEvents() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	eventIDs := make([]string, 0, len(p.queueItems))
	for _, qi := range p.queueItems {
		eventIDs = append(eventIDs, qi.EventID.String())
	}

	if err := p.tx.Where("id IN ?", eventIDs).Find(&p.events).Error; err != nil {
		p.err = fmt.Errorf("failed to load events for merge: %w", err)
		return p
	}

	p.eventsByID = make(map[string]models.WorkflowEvent, len(p.events))
	for _, e := range p.events {
		p.eventsByID[e.ID.String()] = e
	}

	return p
}

func (p *MergeNodeProcessor) selectItemsByParent() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	p.selectedByParent = make(map[string]models.WorkflowNodeQueueItem)
	p.selectedEventByParent = make(map[string]models.WorkflowEvent)

	for _, qi := range p.queueItems {
		e, ok := p.eventsByID[qi.EventID.String()]
		if !ok {
			continue
		}

		parentID := e.NodeID
		if _, req := p.requiredParents[parentID]; !req {
			continue
		}

		if _, already := p.selectedByParent[parentID]; already {
			continue
		}

		p.selectedByParent[parentID] = qi
		p.selectedEventByParent[parentID] = e
	}

	p.coveredParents = make(map[string]struct{}, len(p.selectedByParent))
	for pid := range p.selectedByParent {
		p.coveredParents[pid] = struct{}{}
	}

	return p
}

func (p *MergeNodeProcessor) validateMissingParents() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	if len(p.coveredParents) == len(p.requiredParents) {
		return p
	}

	// Pre-compute channels from each parent to this merge node
	parentAllowedChannels := make(map[string]map[string]struct{})
	for parentID := range p.requiredParents {
		chans := map[string]struct{}{}
		for _, edge := range p.workflow.Edges {
			if edge.SourceID == parentID && edge.TargetID == p.node.NodeID {
				chans[edge.Channel] = struct{}{}
			}
		}
		parentAllowedChannels[parentID] = chans
	}

	// For each missing parent, check if it has finished and emitted no events for channels to this merge
	for parentID := range p.requiredParents {
		if _, ok := p.coveredParents[parentID]; ok {
			continue
		}

		if err := p.validateMissingParent(parentID, parentAllowedChannels[parentID]); err != nil {
			p.err = err
			return p
		}
	}

	// If still not covered, keep waiting
	if len(p.coveredParents) != len(p.requiredParents) {
		p.err = fmt.Errorf("waiting for more parents")
		return p
	}

	return p
}

func (p *MergeNodeProcessor) validateMissingParent(parentID string, allowedChannels map[string]struct{}) error {
	// Find latest execution for this parent and root event
	var parentExec models.WorkflowNodeExecution
	err := p.tx.
		Where("workflow_id = ?", p.node.WorkflowID).
		Where("node_id = ?", parentID).
		Where("root_event_id = ?", p.firstItem.RootEventID).
		Order("created_at DESC").
		Take(&parentExec).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Parent hasn't executed yet
			return fmt.Errorf("parent hasn't executed yet")
		}
		return fmt.Errorf("failed to find parent execution for %s: %w", parentID, err)
	}

	// If not finished, keep waiting
	if parentExec.State != models.WorkflowNodeExecutionStateFinished {
		return fmt.Errorf("parent not finished")
	}

	// If the parent execution failed, this merge can never be fulfilled for this root event.
	// Clean up any queued items for this merge/root and stop processing.
	if parentExec.Result == models.WorkflowNodeExecutionResultFailed {
		if err := p.tx.
			Where("workflow_id = ?", p.node.WorkflowID).
			Where("node_id = ?", p.node.NodeID).
			Where("root_event_id = ?", p.firstItem.RootEventID).
			Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
			return fmt.Errorf("failed to clean merge queue items after parent failure: %w", err)
		}
		p.w.log("Skipping merge for node=%s workflow=%s root_event=%s due to failed parent %s; cleaned pending items",
			p.node.NodeID, p.node.WorkflowID, p.firstItem.RootEventID, parentID)
		return fmt.Errorf("parent failed")
	}

	// Check outputs produced by this parent execution
	outputs, err := parentExec.GetOutputs()
	if err != nil {
		return fmt.Errorf("failed to get outputs for parent %s: %w", parentID, err)
	}

	// If none of the outputs are on channels that go to this merge, exclude this parent
	routedToMerge := false
	for _, ev := range outputs {
		if _, ok := allowedChannels[ev.Channel]; ok {
			routedToMerge = true
			break
		}
	}

	if !routedToMerge {
		// Parent finished but did not route to this merge along any connected channel.
		// This join can never be fulfilled for this root event; clean pending items and stop.
		if err := p.tx.
			Where("workflow_id = ?", p.node.WorkflowID).
			Where("node_id = ?", p.node.NodeID).
			Where("root_event_id = ?", p.firstItem.RootEventID).
			Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
			return fmt.Errorf("failed to clean merge queue items after non-routing parent: %w", err)
		}
		p.w.log("Skipping merge for node=%s workflow=%s root_event=%s due to parent %s not routing to merge; cleaned pending items",
			p.node.NodeID, p.node.WorkflowID, p.firstItem.RootEventID, parentID)
		return fmt.Errorf("parent not routing to merge")
	}

	return nil
}

func (p *MergeNodeProcessor) aggregateInputs() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	p.aggregated = make(map[string]any, len(p.selectedByParent))
	for parentID, ev := range p.selectedEventByParent {
		p.aggregated[parentID] = ev.Data.Data()
	}

	return p
}

func (p *MergeNodeProcessor) createExecution() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	// Create a synthetic input event for the merge execution, set as routed to avoid the router picking it up
	now := time.Now()
	aggEvent := models.WorkflowEvent{
		WorkflowID: p.node.WorkflowID,
		NodeID:     p.node.NodeID, // input for merge
		Channel:    "default",
		Data:       datatypes.NewJSONType(any(p.aggregated)),
		State:      models.WorkflowEventStateRouted,
		CreatedAt:  &now,
	}
	if err := p.tx.Create(&aggEvent).Error; err != nil {
		p.err = fmt.Errorf("failed to create aggregate event for merge: %w", err)
		return p
	}

	// Build configuration with merged input
	config, err := contexts.NewNodeConfigurationBuilder(p.tx, p.node.WorkflowID).
		WithRootEvent(&p.firstItem.RootEventID).
		WithInput(p.aggregated).
		Build(p.node.Configuration.Data())
	if err != nil {
		p.err = err
		return p
	}

	// Create the merge node execution
	nodeExecution := models.WorkflowNodeExecution{
		WorkflowID:    p.node.WorkflowID,
		NodeID:        p.node.NodeID,
		RootEventID:   p.firstItem.RootEventID,
		EventID:       aggEvent.ID,
		State:         models.WorkflowNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(config),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := p.tx.Create(&nodeExecution).Error; err != nil {
		p.err = fmt.Errorf("failed to create merge node execution: %w", err)
		return p
	}

	// Delete the consumed queue items (one per parent)
	for _, qi := range p.selectedByParent {
		if err := qi.Delete(p.tx); err != nil {
			p.err = fmt.Errorf("failed to delete consumed queue item %s: %w", qi.ID, err)
			return p
		}
	}

	// Notify and set node as processing
	messages.NewWorkflowExecutionCreatedMessage(nodeExecution.WorkflowID.String(), &nodeExecution).PublishWithDelay(1 * time.Second)
	if err := p.node.UpdateState(p.tx, models.WorkflowNodeStateProcessing); err != nil {
		p.err = err
		return p
	}

	p.w.log("Created merge execution %s for node=%s workflow=%s with %d inputs", nodeExecution.ID, p.node.NodeID, p.node.WorkflowID, len(p.selectedByParent))
	return p
}

func processMergeNode(tx *gorm.DB, node *models.WorkflowNode, firstItem *models.WorkflowNodeQueueItem) (bool, error) {
	// Note: 'w' needs to be passed in or accessed differently since it's not available in this scope
	// This assumes the WorkflowNodeQueueWorker instance is available
	var w *WorkflowNodeQueueWorker // This needs to be properly injected

	processor := newMergeNodeProcessor(w, tx, node, firstItem)
	return processor.process()
}
