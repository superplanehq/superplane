package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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

type Node struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	RefType       string         `json:"ref_type"`
	Ref           NodeRef        `json:"ref"`
	Configuration map[string]any `json:"configuration"`
}

type NodeRef struct {
	Primitive *PrimitiveRef `json:"primitive"`
	Blueprint *BlueprintRef `json:"blueprint"`
}

type PrimitiveRef struct {
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
