package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

const (
	NodeRefTypeComponent = "component"
	NodeRefTypeBlueprint = "blueprint"

	EdgeTargetTypeNode         = "node"
	EdgeTargetTypeOutputBranch = "output_branch"
)

type Blueprint struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	Nodes          datatypes.JSONSlice[Node]
	Edges          datatypes.JSONSlice[Edge]
	Configuration  datatypes.JSONSlice[components.ConfigurationField]
	OutputBranches datatypes.JSONSlice[components.OutputBranch]
}

func (b *Blueprint) FindNode(id string) (*Node, error) {
	for _, node := range b.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", id)
}

func FindBlueprintByID(id string) (*Blueprint, error) {
	var blueprint Blueprint
	err := database.Conn().
		Where("id = ?", id).
		First(&blueprint).
		Error

	if err != nil {
		return nil, err
	}

	return &blueprint, nil
}

type Node struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	RefType       string         `json:"ref_type"`
	Ref           NodeRef        `json:"ref"`
	Configuration map[string]any `json:"configuration"`
}

type NodeRef struct {
	Component *ComponentRef `json:"component"`
	Blueprint *BlueprintRef `json:"blueprint"`
}

type ComponentRef struct {
	Name string `json:"name"`
}

type BlueprintRef struct {
	ID string `json:"id"`
}

type Edge struct {
	SourceID   string `json:"source_id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Branch     string `json:"branch"`
}
