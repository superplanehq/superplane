package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Workflow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	Edges          datatypes.JSONSlice[Edge]
}

func (w *Workflow) FindNode(id string) (*WorkflowNode, error) {
	var node WorkflowNode
	err := database.Conn().
		Where("workflow_id = ?", w.ID).
		Where("node_id = ?", id).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (w *Workflow) FindNodes() ([]WorkflowNode, error) {
	var nodes []WorkflowNode
	err := database.Conn().
		Where("workflow_id = ?", w.ID).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (w *Workflow) FindEdges(sourceID string, targetType string, channel string) []Edge {
	edges := []Edge{}

	for _, edge := range w.Edges {
		if edge.SourceID == sourceID && edge.TargetType == targetType && edge.Channel == channel {
			edges = append(edges, edge)
		}
	}

	return edges
}

func FindWorkflow(id uuid.UUID) (*Workflow, error) {
	return FindWorkflowInTransaction(database.Conn(), id)
}

func FindWorkflowInTransaction(tx *gorm.DB, id uuid.UUID) (*Workflow, error) {
	var workflow Workflow
	err := tx.
		Where("id = ?", id).
		First(&workflow).
		Error

	if err != nil {
		return nil, err
	}

	return &workflow, nil
}
