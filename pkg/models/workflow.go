package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

const (
	WorkflowNodeExecutionStatePending  = "pending"
	WorkflowNodeExecutionStateWaiting  = "waiting"
	WorkflowNodeExecutionStateStarted  = "started"
	WorkflowNodeExecutionStateFinished = "finished"

	WorkflowNodeExecutionResultPassed = "passed"
	WorkflowNodeExecutionResultFailed = "failed"
)

type Workflow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	Nodes          datatypes.JSONSlice[Node]
	Edges          datatypes.JSONSlice[Edge]
}

func (w *Workflow) FindNode(id string) (*Node, error) {
	for _, node := range w.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", id)
}

type WorkflowQueueItem struct {
	WorkflowID uuid.UUID
	EventID    uuid.UUID
	NodeID     string
	CreatedAt  *time.Time
}

// Returns the oldest queue item for each (workflow,node) pair.
func FindOldestQueueItems() ([]WorkflowQueueItem, error) {
	var items []WorkflowQueueItem

	err := database.Conn().
		Raw(`SELECT DISTINCT ON (workflow_id, node_id) * FROM workflow_queue_items ORDER BY workflow_id, node_id, created_at ASC`).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

func FindWorkflow(id uuid.UUID) (*Workflow, error) {
	var workflow Workflow
	err := database.Conn().
		Where("id = ?", id).
		First(&workflow).
		Error

	if err != nil {
		return nil, err
	}

	return &workflow, nil
}
