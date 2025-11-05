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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func Process(tx *gorm.DB, node *models.WorkflowNode) (bool, error) {
	p := &MergeNodeProcessor{tx: tx, node: node}

	p.loadWorkflow()
	p.findFirstQueueItem()
	p.buildRequiredParents()
	p.fetchMergableQueueItems()
	p.loadEvents()
	p.selectItemsByParent()
	p.validateMissingParents()
	p.aggregateInputs()
	p.buildMergedEvent()
	p.createExecution()
	p.deleteConsumedItems()

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
	aggEvent              *models.WorkflowEvent
}

func (p *MergeNodeProcessor) findFirstQueueItem() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	var queueItem models.WorkflowNodeQueueItem
	err := p.tx.
		Where("workflow_id = ?", p.workflow.ID).
		Where("node_id = ?", p.node.NodeID).
		Order("created_at ASC").
		First(&queueItem).
		Error

	if err != nil {
		p.err = fmt.Errorf("failed to find first queue item for merge: %w", err)
	}

	p.firstItem = &queueItem

	return p
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
		p.log("Merge node %s has no incoming edges; falling back to pass-through", p.node.NodeID)
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

		// After direct-parent checks, run transitive failed-ancestor dominance check.
		if err := p.checkTransitiveFailureDominates(parentID); err != nil {
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

// checkTransitiveFailureDominates walks upstream from the missing direct parent and
// checks if there exists a failed ancestor whose failure dominates all paths from that
// ancestor to the merge (i.e., every path to the merge must pass through the missing parent).
// If such an ancestor is found (based on latest execution for this root event), drop
// the pending merge queue items for this root event and return an error to stop processing.
func (p *MergeNodeProcessor) checkTransitiveFailureDominates(missingParentID string) error {
	rev := map[string][]string{}
	fwd := map[string][]string{}
	for _, e := range p.workflow.Edges {
		rev[e.TargetID] = append(rev[e.TargetID], e.SourceID)
		fwd[e.SourceID] = append(fwd[e.SourceID], e.TargetID)
	}

	// Helper: latest execution failed for a node for this root event?
	latestFailed := func(nodeID string) (bool, error) {
		var exec models.WorkflowNodeExecution
		err := p.tx.
			Where("workflow_id = ?", p.node.WorkflowID).
			Where("node_id = ?", nodeID).
			Where("root_event_id = ?", p.firstItem.RootEventID).
			Order("created_at DESC").
			Take(&exec).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		return exec.State == models.WorkflowNodeExecutionStateFinished && exec.Result == models.WorkflowNodeExecutionResultFailed, nil
	}

	// Helper: does every path from ancestor to this merge pass through missingParentID?
	// BFS from ancestor forward; if we can reach the merge node without visiting missingParentID, then it does not dominate.
	dominates := func(ancestorID string) bool {
		mergeID := p.node.NodeID
		q := []string{ancestorID}
		seen := map[string]struct{}{}
		for len(q) > 0 {
			n := q[0]
			q = q[1:]
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}

			if n == mergeID {
				// Reached merge without passing through missing parent (starting at ancestor which may equal missing parent)
				// Only consider paths that do not include missingParentID; so if n==merge and missing not seen on path, fail dominance.
				if _, ok := seen[missingParentID]; !ok && ancestorID != missingParentID {
					return false
				}
			}

			for _, nxt := range fwd[n] {
				if nxt == missingParentID {
					// Paths through missing parent are allowed; continue exploring beyond it
					// but mark it seen so merge detection above knows it was traversed.
				}
				q = append(q, nxt)
			}
		}
		// If every way to merge necessarily touches missingParentID, then ancestor dominates.
		return true
	}

	// Walk ancestors transitively and check for failed ones that dominate.
	stack := []string{missingParentID}
	visited := map[string]struct{}{}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := visited[cur]; ok {
			continue
		}
		visited[cur] = struct{}{}

		// For each ancestor of cur
		for _, anc := range rev[cur] {
			failed, err := latestFailed(anc)
			if err != nil {
				return fmt.Errorf("failed checking ancestor %s: %w", anc, err)
			}
			if failed && dominates(anc) {
				if err := p.tx.
					Where("workflow_id = ?", p.node.WorkflowID).
					Where("node_id = ?", p.node.NodeID).
					Where("root_event_id = ?", p.firstItem.RootEventID).
					Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
					return fmt.Errorf("failed to clean merge queue items after transitive failed ancestor %s: %w", anc, err)
				}
				return fmt.Errorf("unfulfillable due to failed ancestor %s", anc)
			}

			stack = append(stack, anc)
		}
	}

	return nil
}

func (p *MergeNodeProcessor) validateMissingParent(parentID string, allowedChannels map[string]struct{}) error {
	var parentExec models.WorkflowNodeExecution

	err := p.tx.
		Where("workflow_id = ?", p.node.WorkflowID).
		Where("node_id = ?", parentID).
		Where("root_event_id = ?", p.firstItem.RootEventID).
		Order("created_at DESC").
		Take(&parentExec).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Parent hasn't executed yet; allow transitive check to decide
			return nil
		}

		return fmt.Errorf("failed to find parent execution for %s: %w", parentID, err)
	}

	if parentExec.State != models.WorkflowNodeExecutionStateFinished {
		// Not finished yet; allow transitive check and overall wait handling
		return nil
	}

	if parentExec.Result == models.WorkflowNodeExecutionResultFailed {
		// If the parent execution failed, this merge can never be fulfilled for this root event.
		// Clean up any queued items for this merge/root and stop processing.
		err := p.tx.
			Where("workflow_id = ?", p.node.WorkflowID).
			Where("node_id = ?", p.node.NodeID).
			Where("root_event_id = ?", p.firstItem.RootEventID).
			Delete(&models.WorkflowNodeQueueItem{}).Error

		if err != nil {
			return fmt.Errorf("failed to clean merge queue items after parent failure: %w", err)
		}

		p.log("Skipping merge for node=%s workflow=%s root_event=%s due to failed parent %s; cleaned pending items", p.node.NodeID, p.node.WorkflowID, p.firstItem.RootEventID, parentID)
		return fmt.Errorf("parent failed")
	}

	outputs, err := parentExec.GetOutputs()
	if err != nil {
		return fmt.Errorf("failed to get outputs for parent %s: %w", parentID, err)
	}

	routedToMerge := false
	for _, ev := range outputs {
		if _, ok := allowedChannels[ev.Channel]; ok {
			routedToMerge = true
			break
		}
	}

	if !routedToMerge {
		err := p.tx.
			Where("workflow_id = ?", p.node.WorkflowID).
			Where("node_id = ?", p.node.NodeID).
			Where("root_event_id = ?", p.firstItem.RootEventID).
			Delete(&models.WorkflowNodeQueueItem{}).Error

		if err != nil {
			return fmt.Errorf("failed to clean merge queue items after non-routing parent: %w", err)
		}

		p.log("Skipping merge for node=%s workflow=%s root_event=%s due to parent %s not routing to merge; cleaned pending items", p.node.NodeID, p.node.WorkflowID, p.firstItem.RootEventID, parentID)

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

func (p *MergeNodeProcessor) buildMergedEvent() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	now := time.Now()

	aggEvent := models.WorkflowEvent{
		WorkflowID: p.node.WorkflowID,
		NodeID:     p.node.NodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType(any(p.aggregated)),
		State:      models.WorkflowEventStateRouted,
		CreatedAt:  &now,
	}

	err := p.tx.Create(&aggEvent).Error

	if err != nil {
		p.err = fmt.Errorf("failed to create aggregate event for merge: %w", err)
		return p
	}

	p.aggEvent = &aggEvent

	return p
}

func (p *MergeNodeProcessor) createExecution() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	if p.aggEvent == nil {
		p.err = fmt.Errorf("aggregate event not built")
		return p
	}
	config, err := contexts.NewNodeConfigurationBuilder(p.tx, p.node.WorkflowID).
		WithRootEvent(&p.firstItem.RootEventID).
		WithInput(p.aggregated).
		Build(p.node.Configuration.Data())

	if err != nil {
		p.err = err
		return p
	}

	now := time.Now()

	nodeExecution := models.WorkflowNodeExecution{
		WorkflowID:    p.node.WorkflowID,
		NodeID:        p.node.NodeID,
		RootEventID:   p.firstItem.RootEventID,
		EventID:       p.aggEvent.ID,
		State:         models.WorkflowNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(config),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	err = p.tx.Create(&nodeExecution).Error

	if err != nil {
		p.err = fmt.Errorf("failed to create merge node execution: %w", err)
		return p
	}

	messages.NewWorkflowExecutionCreatedMessage(nodeExecution.WorkflowID.String(), &nodeExecution).PublishWithDelay(1 * time.Second)
	if err := p.node.UpdateState(p.tx, models.WorkflowNodeStateProcessing); err != nil {
		p.err = err
		return p
	}

	p.log("Created merge execution %s for node=%s workflow=%s with %d inputs", nodeExecution.ID, p.node.NodeID, p.node.WorkflowID, len(p.selectedByParent))
	return p
}

func (p *MergeNodeProcessor) deleteConsumedItems() *MergeNodeProcessor {
	if p.err != nil {
		return p
	}

	for _, qi := range p.selectedByParent {
		if err := qi.Delete(p.tx); err != nil {
			p.err = fmt.Errorf("failed to delete consumed queue item %s: %w", qi.ID, err)
			return p
		}
	}

	return p
}

func (p *MergeNodeProcessor) log(format string, args ...any) {
	fmt.Printf("[MergeNodeProcessor] "+format+"\n", args...)
}
