package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	NodeRefTypeComponent = "component"
	NodeRefTypeBlueprint = "blueprint"
	NodeRefTypeTrigger   = "trigger"

	EdgeTargetTypeNode          = "node"
	EdgeTargetTypeOutputChannel = "output-channel"
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
	OutputChannels datatypes.JSONSlice[components.OutputChannel]
}

func (b *Blueprint) FindNode(id string) (*Node, error) {
	for _, node := range b.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", id)
}

func (b *Blueprint) FindEdges(sourceID string, targetType string, channel string) []Edge {
	edges := []Edge{}

	for _, edge := range b.Edges {
		if edge.SourceID == sourceID && edge.TargetType == targetType && edge.Channel == channel {
			edges = append(edges, edge)
		}
	}

	return edges
}

func (b *Blueprint) OutputChannelEdges() []Edge {
	edges := []Edge{}
	for _, edge := range b.Edges {
		if edge.TargetType == EdgeTargetTypeOutputChannel {
			edges = append(edges, edge)
		}
	}

	return edges
}

// TODO: this is where input channels come in
func (b *Blueprint) FindRootNode() *Node {
	hasIncoming := make(map[string]bool)
	for _, edge := range b.Edges {
		if edge.TargetType == EdgeTargetTypeNode {
			hasIncoming[edge.TargetID] = true
		}
	}

	for _, node := range b.Nodes {
		if !hasIncoming[node.ID] {
			return &node
		}
	}

	return nil
}

func FindBlueprintByID(id string) (*Blueprint, error) {
	return FindBlueprintByIDInTransaction(database.Conn(), id)
}

func FindBlueprintByIDInTransaction(tx *gorm.DB, id string) (*Blueprint, error) {
	var blueprint Blueprint
	err := tx.
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
	Component *ComponentRef `json:"component,omitempty"`
	Blueprint *BlueprintRef `json:"blueprint,omitempty"`
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
	Channel    string `json:"channel"`
}
