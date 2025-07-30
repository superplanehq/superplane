package models

import (
	"encoding/json"
	"time"
)

const (
	ExecutionFinishedEventType = "execution_finished"
	FieldSetCompletedEventType = "field_set_completed"
)

//
// Execution finished event
//

type ExecutionFinishedEvent struct {
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

//
// Field set completed event
//

type FieldSetCompletedEvent struct {
	Type    string            `json:"type"`
	Fields  map[string]string `json:"fields"`
	Events  map[string]any    `json:"events"`
	Missing []string          `json:"missing,omitempty"`
}

func NewExecutionCompletionEvent(execution *StageExecution, outputs map[string]any) (*ExecutionFinishedEvent, error) {
	return &ExecutionFinishedEvent{
		Type: ExecutionFinishedEventType,
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

func NewFieldSetCompletedEvent(fields map[string]string, events []ConnectionGroupFieldSetEventWithData, missingConnections []Connection) (*FieldSetCompletedEvent, error) {
	eventMap := map[string]any{}
	for _, e := range events {
		var obj map[string]any
		err := json.Unmarshal(e.Raw, &obj)
		if err != nil {
			return nil, err
		}

		eventMap[e.SourceName] = obj
	}

	e := FieldSetCompletedEvent{
		Type:    FieldSetCompletedEventType,
		Fields:  fields,
		Events:  eventMap,
		Missing: nil,
	}

	if len(missingConnections) == 0 {
		return &e, nil
	}

	//
	// Include the missing field, if any.
	//
	e.Missing = []string{}
	for _, connection := range missingConnections {
		e.Missing = append(e.Missing, connection.SourceName)
	}

	return &e, nil
}
