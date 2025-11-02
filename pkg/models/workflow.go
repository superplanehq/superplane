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
	CreatedBy      *uuid.UUID
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

func FindWorkflowNodes(workflowID uuid.UUID) ([]WorkflowNode, error) {
	var nodes []WorkflowNode
	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (w *Workflow) FindEdges(sourceID string, channel string) []Edge {
	edges := []Edge{}

	for _, edge := range w.Edges {
		if edge.SourceID == sourceID && edge.Channel == channel {
			edges = append(edges, edge)
		}
	}

	return edges
}

func FindWorkflow(orgID, id uuid.UUID) (*Workflow, error) {
	return FindWorkflowInTransaction(database.Conn(), orgID, id)
}

func FindWorkflowByName(name string, organizationID uuid.UUID) (*Workflow, error) {
	var workflow Workflow
	err := database.Conn().
		Where("name = ? AND organization_id = ?", name, organizationID).
		First(&workflow).
		Error

	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

func FindWorkflowInTransaction(tx *gorm.DB, orgID, id uuid.UUID) (*Workflow, error) {
	var workflow Workflow
	err := tx.
		Where("organization_id = ?", orgID).
		Where("id = ?", id).
		First(&workflow).
		Error

	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

func FindUnscopedWorkflow(id uuid.UUID) (*Workflow, error) {
	return FindUnscopedWorkflowInTransaction(database.Conn(), id)
}

func FindUnscopedWorkflowInTransaction(tx *gorm.DB, id uuid.UUID) (*Workflow, error) {
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
