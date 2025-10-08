package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

const (
	NodeRefTypeComponent = "component"
	NodeRefTypeBlueprint = "blueprint"
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
}

func (b *Blueprint) FindNode(id string) (*Node, error) {
	for _, node := range b.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", id)
}

func FindBlueprintByName(name string) (*Blueprint, error) {
	var blueprint Blueprint
	err := database.Conn().
		Where("name = ?", name).
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
	Name string `json:"name"`
}

type Edge struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Branch   string `json:"branch"`
}
