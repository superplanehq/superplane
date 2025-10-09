package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
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
