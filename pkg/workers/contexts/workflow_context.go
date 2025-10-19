package contexts

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type WorkflowContext struct {
	tx        *gorm.DB
	execution *models.WorkflowNodeExecution
}

func NewWorkflowContext(tx *gorm.DB, execution *models.WorkflowNodeExecution) components.WorkflowContext {
	return &WorkflowContext{tx: tx, execution: execution}
}

func (c *WorkflowContext) SourceNode() (*components.WorkflowNode, error) {
	event, err := models.FindWorkflowEventInTransaction(c.tx, c.execution.EventID)
	if err != nil {
		return nil, err
	}

	return &components.WorkflowNode{ID: event.NodeID}, nil
}

func (c *WorkflowContext) PreviousNodes() ([]components.WorkflowNode, error) {
	workflow, err := models.FindUnscopedWorkflowInTransaction(c.tx, c.execution.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("error finding workflow: %v", err)
	}

	previousNodes := []components.WorkflowNode{}
	for _, edge := range workflow.Edges {
		if edge.TargetID != c.execution.NodeID {
			continue
		}

		previousNodes = append(previousNodes, components.WorkflowNode{ID: edge.SourceID})
	}

	return previousNodes, nil
}

func (c *WorkflowContext) Dequeue() (*components.NodeQueueItem, error) {
	return nil, fmt.Errorf("not implemented yet")
}
