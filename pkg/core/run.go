package core

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const (
	RunResultPassed    = "passed"
	RunResultFailed    = "failed"
	RunResultCancelled = "cancelled"
)

type RunExecutionContext interface {
	Create(params RunCreationParams) (*Run, error)
	Cancel() error
	AssignOutput(output map[string]any) error
	AddError(message string) error
}

type Run struct {
	ID     uuid.UUID `json:"id" mapstructure:"id"`
	AppID  uuid.UUID `json:"app_id" mapstructure:"app_id"`
	Result string    `json:"result" mapstructure:"result"`
	Errors []string  `json:"errors" mapstructure:"errors"`
}

type RunCreationParams struct {
	Input     any
	App       string
	Node      string
	Callbacks []RunCallback
}

const (
	RunCallbackWhenPending  = "pending"
	RunCallbackWhenFinished = "finished"

	RunCallbackOnEntry  = "entry"
	RunCallbackOnParent = "parent"
)

/*
 * Run callbacks are the way for components
 * to execute custom behavior during a run's lifecycle.
 */
type RunCallback struct {
	When string `json:"when" mapstructure:"when"`
	On   string `json:"on" mapstructure:"on"`
	Hook string `json:"hook" mapstructure:"hook"`
}

/*
 * RunFinishedCallback is the payload used in the RunCallbackWhenFinished run lifecycle callback.
 */
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
