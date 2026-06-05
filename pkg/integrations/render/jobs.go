package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CreateJobPayloadType = "render.job.created"
	GetJobPayloadType    = "render.job"
)

type CreateJob struct{}
type GetJob struct{}

type CreateJobConfiguration struct {
	Service      string `json:"service" mapstructure:"service"`
	StartCommand string `json:"startCommand" mapstructure:"startCommand"`
	PlanID       string `json:"planId" mapstructure:"planId"`
}

type GetJobConfiguration struct {
	Service string `json:"service" mapstructure:"service"`
	JobID   string `json:"jobId" mapstructure:"jobId"`
}

func (c *CreateJob) Name() string        { return "render.createJob" }
func (c *CreateJob) Label() string       { return "Create Job" }
func (c *CreateJob) Description() string { return "Create a Render one-off job" }
func (c *CreateJob) Documentation() string {
	return `Create a Render one-off job from an existing service. Use it for health checks, queue drains, and repair tasks.`
}
func (c *CreateJob) Icon() string  { return "terminal" }
func (c *CreateJob) Color() string { return "gray" }
func (c *CreateJob) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *CreateJob) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceField("Render service to run the job from"),
		{Name: "startCommand", Label: "Start Command", Type: configuration.FieldTypeString, Required: true, Description: "Command the one-off job should run"},
		{Name: "planId", Label: "Plan ID", Type: configuration.FieldTypeString, Required: false, Description: "Optional Render plan ID for the job"},
	}
}

func (c *GetJob) Name() string        { return "render.getJob" }
func (c *GetJob) Label() string       { return "Get Job" }
func (c *GetJob) Description() string { return "Retrieve a Render one-off job" }
func (c *GetJob) Documentation() string {
	return `Retrieve the current status of a Render one-off job.`
}
func (c *GetJob) Icon() string  { return "terminal" }
func (c *GetJob) Color() string { return "gray" }
func (c *GetJob) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *GetJob) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceField("Render service that owns the job"),
		{Name: "jobId", Label: "Job ID", Type: configuration.FieldTypeString, Required: true, Description: "Render job ID"},
	}
}

func decodeCreateJobConfiguration(configuration any) (CreateJobConfiguration, error) {
	spec := CreateJobConfiguration{}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return CreateJobConfiguration{}, err
	}
	spec.Service = strings.TrimSpace(spec.Service)
	spec.StartCommand = strings.TrimSpace(spec.StartCommand)
	spec.PlanID = strings.TrimSpace(spec.PlanID)
	if spec.Service == "" {
		return CreateJobConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.StartCommand == "" {
		return CreateJobConfiguration{}, fmt.Errorf("startCommand is required")
	}
	return spec, nil
}

func decodeGetJobConfiguration(configuration any) (GetJobConfiguration, error) {
	spec := GetJobConfiguration{}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return GetJobConfiguration{}, err
	}
	spec.Service = strings.TrimSpace(spec.Service)
	spec.JobID = strings.TrimSpace(spec.JobID)
	if spec.Service == "" {
		return GetJobConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.JobID == "" {
		return GetJobConfiguration{}, fmt.Errorf("jobId is required")
	}
	return spec, nil
}

func (c *CreateJob) Setup(ctx core.SetupContext) error {
	_, err := decodeCreateJobConfiguration(ctx.Configuration)
	return err
}

func (c *CreateJob) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateJobConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	job, err := client.CreateJob(spec.Service, spec.StartCommand, spec.PlanID)
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateJobPayloadType, []any{jobData(job)})
}

func (c *GetJob) Setup(ctx core.SetupContext) error {
	_, err := decodeGetJobConfiguration(ctx.Configuration)
	return err
}

func (c *GetJob) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetJobConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	job, err := client.GetJob(spec.Service, spec.JobID)
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetJobPayloadType, []any{jobData(job)})
}

func jobData(job JobResponse) map[string]any {
	data := map[string]any{
		"jobId":        job.ID,
		"serviceId":    job.ServiceID,
		"startCommand": job.StartCommand,
	}
	if job.PlanID != "" {
		data["planId"] = job.PlanID
	}
	if job.Status != "" {
		data["status"] = job.Status
	}
	if job.CreatedAt != "" {
		data["createdAt"] = job.CreatedAt
	}
	if job.StartedAt != "" {
		data["startedAt"] = job.StartedAt
	}
	if job.FinishedAt != "" {
		data["finishedAt"] = job.FinishedAt
	}
	return data
}

func (c *CreateJob) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *CreateJob) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *CreateJob) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *CreateJob) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *CreateJob) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *CreateJob) HandleHook(ctx core.ActionHookContext) error { return nil }

func (c *GetJob) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *GetJob) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *GetJob) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *GetJob) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *GetJob) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *GetJob) HandleHook(ctx core.ActionHookContext) error { return nil }
