package core

import "github.com/google/uuid"

const (
	RunResultPassed = "passed"
	RunResultFailed = "failed"

	RunCallbackKindInit     = "init"
	RunCallbackKindFinished = "finished"

	RunCallbackRefTarget = "target"
	RunCallbackRefParent = "parent"
)

type RunExecutionContext interface {
	Create(params RunCreationParams) (*Run, error)
}

type Run struct {
	ID     uuid.UUID `json:"id" mapstructure:"id"`
	Result string    `json:"result" mapstructure:"result"`
	Error  *string   `json:"error,omitempty" mapstructure:"error,omitempty"`
}

type RunCreationParams struct {
	Input     any
	App       string
	Node      string
	Callbacks []RunCallbackDefinition
}

type RunCallbackDefinition struct {
	Kind string
	Ref  string
	Hook string
}
