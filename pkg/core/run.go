package core

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

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
	AppID  uuid.UUID `json:"app_id" mapstructure:"app_id"`
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

// RunFinishedCallback is the payload for RunCallbackKindFinished hooks on the parent.
type RunFinishedCallback struct {
	Run Run `json:"run" mapstructure:"run"`
}

func NewRunFinishedCallback(run Run) RunFinishedCallback {
	return RunFinishedCallback{Run: run}
}

func (c RunFinishedCallback) ToParameters() (map[string]any, error) {
	return runCallbackToParameters(c)
}

func DecodeRunFinishedCallback(params map[string]any) (RunFinishedCallback, error) {
	return decodeRunCallback[RunFinishedCallback](params)
}

func runCallbackToParameters(payload any) (map[string]any, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode run callback: %w", err)
	}

	var parameters map[string]any
	if err := json.Unmarshal(data, &parameters); err != nil {
		return nil, fmt.Errorf("encode run callback: %w", err)
	}

	return parameters, nil
}

func decodeRunCallback[T any](params map[string]any) (T, error) {
	var callback T

	data, err := json.Marshal(params)
	if err != nil {
		return callback, fmt.Errorf("decode run callback: %w", err)
	}

	if err := json.Unmarshal(data, &callback); err != nil {
		return callback, fmt.Errorf("decode run callback: %w", err)
	}

	return callback, nil
}

// NewRun builds the run snapshot carried in run callback payloads.
func NewRun(id, appID uuid.UUID, result string, errMessage *string) Run {
	return Run{
		ID:     id,
		AppID:  appID,
		Result: result,
		Error:  errMessage,
	}
}
