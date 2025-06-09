package models

import (
	"time"
)

//
// This is the event that is emitted when a stage finishes its execution.
//

const (
	StageExecutionCompletionType = "StageExecutionCompletion"
)

type StageExecutionCompletion struct {
	Type      string            `json:"type"`
	Stage     *StageInEvent     `json:"stage,omitempty"`
	Execution *ExecutionInEvent `json:"execution,omitempty"`
	Outputs   map[string]any    `json:"outputs,omitempty"`
}

type StageInEvent struct {
	ID string `json:"id"`
}

type ExecutionInEvent struct {
	ID         string     `json:"id"`
	Result     string     `json:"result"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func NewStageExecutionCompletion(execution *StageExecution, outputs map[string]any) (*StageExecutionCompletion, error) {
	return &StageExecutionCompletion{
		Type: StageExecutionCompletionType,
		Stage: &StageInEvent{
			ID: execution.StageID.String(),
		},
		Execution: &ExecutionInEvent{
			ID:         execution.ID.String(),
			Result:     execution.Result,
			CreatedAt:  execution.CreatedAt,
			StartedAt:  execution.StartedAt,
			FinishedAt: execution.FinishedAt,
		},
		Outputs: outputs,
	}, nil
}
