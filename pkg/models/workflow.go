package models

import (
	"time"

	"github.com/google/uuid"
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

type WorkflowQueue struct {
	WorkflowID uuid.UUID
	NodeID     string
	Data       datatypes.JSONType[any]
}

type WorkflowNodeExecution struct {
	WorkflowID    uuid.UUID
	NodeID        string
	State         string
	Result        string
	ResultReason  string
	ResultMessage string
	Input         datatypes.JSONType[any]
	Output        datatypes.JSONType[any]
}
