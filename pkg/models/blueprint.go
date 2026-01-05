package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Blueprint struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	Icon           string
	Color          string
	CreatedBy      *uuid.UUID
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	Nodes          datatypes.JSONSlice[Node]
	Edges          datatypes.JSONSlice[Edge]
	Configuration  datatypes.JSONSlice[configuration.Field]
	OutputChannels datatypes.JSONSlice[BlueprintOutputChannel]
}

type BlueprintOutputChannel struct {
	Name              string `json:"name"`
	NodeID            string `json:"node_id"`
	NodeOutputChannel string `json:"node_output_channel"`
}

func (b *Blueprint) FindNode(id string) (*Node, error) {
	for _, node := range b.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", id)
}

func (b *Blueprint) FindEdges(sourceID string, channel string) []Edge {
	edges := []Edge{}

	for _, edge := range b.Edges {
		if edge.SourceID == sourceID && edge.Channel == channel {
			edges = append(edges, edge)
		}
	}

	return edges
}

func (b *Blueprint) FindRootNode() *Node {
	hasIncoming := make(map[string]bool)
	for _, edge := range b.Edges {
		hasIncoming[edge.TargetID] = true
	}

	for _, node := range b.Nodes {
		if !hasIncoming[node.ID] {
			return &node
		}
	}

	return nil
}

func FindBlueprint(orgID, id string) (*Blueprint, error) {
	var blueprint Blueprint
	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("id = ?", id).
		First(&blueprint).
		Error

	if err != nil {
		return nil, err
	}

	return &blueprint, nil
}

func FindBlueprintByName(name string, orgID uuid.UUID) (*Blueprint, error) {
	var blueprint Blueprint
	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("name = ?", name).
		First(&blueprint).
		Error

	if err != nil {
		return nil, err
	}

	return &blueprint, nil
}

func FindUnscopedBlueprint(id string) (*Blueprint, error) {
	return FindUnscopedBlueprintInTransaction(database.Conn(), id)
}

func FindUnscopedBlueprintInTransaction(tx *gorm.DB, id string) (*Blueprint, error) {
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
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Type              string         `json:"type"`
	Ref               NodeRef        `json:"ref"`
	Configuration     map[string]any `json:"configuration"`
	Metadata          map[string]any `json:"metadata"`
	Position          Position       `json:"position"`
	IsCollapsed       bool           `json:"isCollapsed"`
	AppInstallationID *string        `json:"appInstallationId,omitempty"`
	ErrorMessage      *string        `json:"errorMessage,omitempty"`
	AnnotationText    *string        `json:"annotationText,omitempty"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type NodeRef struct {
	Component *ComponentRef `json:"component,omitempty"`
	Blueprint *BlueprintRef `json:"blueprint,omitempty"`
	Trigger   *TriggerRef   `json:"trigger,omitempty"`
	Widget    *WidgetRef    `json:"widget,omitempty"`
}

type ComponentRef struct {
	Name string `json:"name"`
}

type TriggerRef struct {
	Name string `json:"name"`
}

type WidgetRef struct {
	Name string `json:"name"`
}

type BlueprintRef struct {
	ID string `json:"id"`
}

type Edge struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Channel  string `json:"channel"`
}
