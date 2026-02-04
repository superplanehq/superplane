package semaphore

import (
	"fmt"

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
	return "Fetch logs for a specific Semaphore job"
}

func (g *GetJobLogs) Documentation() string {
	return `The Get Job Logs component retrieves the full log output for a specific Semaphore job.

## Configuration
- **Job ID**: The unique identifier for the job.

## Output Channels
- **Done**: Emitted when logs are retrieved, containing the log content.`
}

func (g *GetJobLogs) Icon() string {
	return "list"
}

func (g *GetJobLogs) Color() string {
	return "gray"
}

func (g *GetJobLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  "done",
			Label: "Done",
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

func (g *GetJobLogs) Execute(ctx core.ExecutionContext) error {
	spec := GetJobLogsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	logs, err := client.GetJobLogs(spec.JobID)
	if err != nil {
		return fmt.Errorf("error fetching job logs %s: %v", spec.JobID, err)
	}

	return ctx.ExecutionState.Emit("done", "semaphore.job.logs.fetched", logs)
}

func (g *GetJobLogs) Setup(ctx core.SetupContext) error                          { return nil }
func (g *GetJobLogs) Cancel(ctx core.ExecutionContext) error                     { return nil }
func (g *GetJobLogs) Cleanup(ctx core.SetupContext) error                        { return nil }
func (g *GetJobLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error)  { return (200, nil) }
func (g *GetJobLogs) Actions() []core.Action                                     { return nil }
func (g *GetJobLogs) HandleAction(ctx core.ActionContext) error                  { return nil }
func (g *GetJobLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
