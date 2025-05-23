package events

import (
	"time"

	"github.com/superplanehq/superplane/pkg/models"
)

//
// This is the event that is emitted when a stage finishes its execution.
//

const (
	StageExecutionCompletionType = "StageExecutionCompletion"
)

type StageExecutionCompletion struct {
	Type      string            `json:"type"`
	Stage     *Stage            `json:"stage,omitempty"`
	Execution *Execution        `json:"execution,omitempty"`
	KV        map[string]string `json:"kv,omitempty"`
}

type Stage struct {
	ID string `json:"id"`
}

type Execution struct {
	ID         string     `json:"id"`
	Result     string     `json:"result"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func NewStageExecutionCompletion(execution *models.StageExecution, kv map[string]string) (*StageExecutionCompletion, error) {
	return &StageExecutionCompletion{
		Type: StageExecutionCompletionType,
		Stage: &Stage{
			ID: execution.StageID.String(),
		},
		Execution: &Execution{
			ID:         execution.ID.String(),
			Result:     execution.Result,
			CreatedAt:  execution.CreatedAt,
			StartedAt:  execution.StartedAt,
			FinishedAt: execution.FinishedAt,
		},
		KV: kv,
	}, nil
}
