package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetJobLogs struct{}

type GetJobLogsSpec struct {
	JobID string `json:"jobId"`
}

func (g *GetJobLogs) Name() string {
	return "semaphore.getJobLogs"
}

func (g *GetJobLogs) Label() string {
	return "Get Job Logs"
}

func (g *GetJobLogs) Description() string {
	return "Fetch log output for a Semaphore job"
}

func (g *GetJobLogs) Documentation() string {
	return `The Get Job Logs component retrieves the full log output for a specific Semaphore job.

## Configuration

- **Job ID**: The unique identifier of the Semaphore job to fetch logs for.

## Outputs

- **Output**: Emits the raw log content as a string.`
}

func (g *GetJobLogs) Icon() string {
	return "file-text"
}

func (g *GetJobLogs) Color() string {
	return "gray"
}

func (g *GetJobLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  "output",
			Label: "Output",
		},
	}
}

func (g *GetJobLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "jobId",
			Label:       "Job ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the job to fetch logs for",
		},
	}
}

func (g *GetJobLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetJobLogs) Setup(ctx core.SetupContext) error {
	spec := GetJobLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if spec.JobID == "" {
		return fmt.Errorf("jobId is required")
	}

	return nil
}

func (g *GetJobLogs) ExampleOutput() map[string]any {
	return map[string]any{
		"output": "job logs content...",
	}
}

func (g *GetJobLogs) Execute(ctx core.ExecutionContext) error {
	spec := GetJobLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	logs, err := client.GetJobLogs(spec.JobID)
	if err != nil {
		return fmt.Errorf("error fetching job logs: %v", err)
	}

	return ctx.ExecutionState.Emit("output", "semaphore.job.logs", []any{string(logs)})
}

func (g *GetJobLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (g *GetJobLogs) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetJobLogs) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetJobLogs) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetJobLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}
