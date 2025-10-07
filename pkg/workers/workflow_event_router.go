package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type WorkflowEventRouter struct{}

func NewWorkflowEventRouter() *WorkflowEventRouter {
	return &WorkflowEventRouter{}
}

func (w *WorkflowEventRouter) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processEvents(); err != nil {
				log.Printf("Error processing workflow events: %v", err)
			}
		}
	}
}

func (w *WorkflowEventRouter) processEvents() error {
	events, err := models.ListEventsToRoute()
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := w.routeEvent(&event); err != nil {
			log.Printf("Error routing event %s: %v", event.ID, err)
			if err := event.Fail(); err != nil {
				log.Printf("Error marking event %s as failed: %v", event.ID, err)
			}
		}
	}

	return nil
}

func (w *WorkflowEventRouter) routeEvent(event *models.WorkflowEvent) error {
	nodes, edges, err := w.findNodesAndEdges(event)
	if err != nil {
		return fmt.Errorf("failed to determine nodes and edges for event: %v", err)
	}

	nextNodeID, err := w.findNextNode(event.ID, nodes, edges)
	if err != nil {
		return fmt.Errorf("failed to find next node: %w", err)
	}

	//
	// No more nodes to execute, complete workflow event.
	//
	if nextNodeID == "" {
		log.Printf("[WorkflowEventRouter] Event %s: no more nodes, completing event", event.ID)
		if event.ParentEventID != nil {
			return w.completeBlueprintExecution(event)
		}

		return event.Complete()
	}

	log.Printf("[WorkflowEventRouter] Event %s: routing to node %s", event.ID, nextNodeID)

	//
	// Create queue entry for next node and
	// move workflow event to 'processing' state.
	//
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		queueEntry := &models.WorkflowQueueItem{
			WorkflowID: event.WorkflowID,
			EventID:    event.ID,
			NodeID:     nextNodeID,
			CreatedAt:  &now,
		}

		err := tx.Create(queueEntry).Error
		if err != nil {
			return err
		}

		return event.ProcessingInTransaction(tx)
	})
}

func (w *WorkflowEventRouter) findNodesAndEdges(event *models.WorkflowEvent) ([]models.Node, []models.Edge, error) {
	//
	// If this event is for a blueprint, load the blueprint structure
	//
	if event.BlueprintName != nil {
		log.Printf("[WorkflowEventRouter] Event %s: routing through blueprint '%s'", event.ID, *event.BlueprintName)
		blueprint, err := models.FindBlueprintByName(*event.BlueprintName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find blueprint %s: %w", *event.BlueprintName, err)
		}

		return blueprint.Nodes, blueprint.Edges, nil
	}

	//
	// Otherwise, load the workflow structure.
	//
	log.Printf("[WorkflowEventRouter] Event %s: routing through workflow %s", event.ID, event.WorkflowID)
	workflow, err := models.FindWorkflow(event.WorkflowID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find workflow %s: %w", event.WorkflowID, err)
	}

	return workflow.Nodes, workflow.Edges, nil
}

func (w *WorkflowEventRouter) findNextNode(workflowEventID uuid.UUID, nodes []models.Node, edges []models.Edge) (string, error) {

	//
	// Find the last execution for this workflow event.
	//
	lastExecution, err := models.FindLastNodeExecutionForEvent(workflowEventID, []string{models.WorkflowNodeExecutionStateFinished})

	//
	// If no previous execution exists,
	// we need to start with the first node.
	//
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return w.findStartNode(nodes, edges)
	}

	if err != nil {
		return "", err
	}

	// Find outgoing edges from last executed node
	var outgoingEdges []models.Edge
	for _, edge := range edges {
		if edge.SourceID == lastExecution.NodeID {
			outgoingEdges = append(outgoingEdges, edge)
		}
	}

	//
	// If no outgoing edges exist,
	// we have reached the end of the workflow event chain.
	//
	if len(outgoingEdges) == 0 {
		return "", nil
	}

	// Check which branch has data in outputs
	for _, edge := range outgoingEdges {
		if edge.Branch == "" {
			// No branch specified, use this edge
			return edge.TargetID, nil
		}

		outputs := lastExecution.Outputs.Data()
		if _, exists := outputs[edge.Branch]; exists {
			return edge.TargetID, nil
		}
	}

	return "", nil
}

func (w *WorkflowEventRouter) findStartNode(nodes []models.Node, edges []models.Edge) (string, error) {
	//
	// Find nodes with no incoming edges
	// TODO: this should somehow be connected to "triggers" / "event sources"
	//
	hasIncoming := make(map[string]bool)
	for _, edge := range edges {
		hasIncoming[edge.TargetID] = true
	}

	for _, node := range nodes {
		if !hasIncoming[node.ID] {
			return node.ID, nil
		}
	}

	return "", nil
}

func (w *WorkflowEventRouter) completeBlueprintExecution(childEvent *models.WorkflowEvent) error {
	log.Printf("[WorkflowEventRouter] Completing blueprint execution for child event %s", childEvent.ID)

	parentEvent, err := models.FindWorkflowEvent(childEvent.ParentEventID.String())
	if err != nil {
		return fmt.Errorf("failed to find parent event %s: %w", *childEvent.ParentEventID, err)
	}

	var execution models.WorkflowNodeExecution
	err = database.Conn().
		Where("workflow_id = ?", parentEvent.WorkflowID).
		Where("event_id = ?", parentEvent.ID).
		Where("state = ?", models.WorkflowNodeExecutionStateStarted).
		First(&execution).
		Error

	if err != nil {
		return fmt.Errorf("failed to find blueprint node execution: %w", err)
	}

	log.Printf("[WorkflowEventRouter] Completing blueprint node execution: workflow=%s, node=%s", execution.WorkflowID, execution.NodeID)

	//
	// If this is a child event (blueprint execution), we need to:
	// 1. Complete the child event
	// 2. Complete the parent blueprint node execution
	// 3. Move the parent event back to routing
	//
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := childEvent.Complete(); err != nil {
			return fmt.Errorf("failed to complete child event: %w", err)
		}

		if err := execution.Pass(map[string][]any{}); err != nil {
			return fmt.Errorf("failed to pass blueprint node execution: %w", err)
		}

		if err := parentEvent.Route(); err != nil {
			return fmt.Errorf("failed to route parent event: %w", err)
		}

		log.Printf("[WorkflowEventRouter] Blueprint execution completed, parent event %s moved back to routing", parentEvent.ID)
		return nil
	})
}
