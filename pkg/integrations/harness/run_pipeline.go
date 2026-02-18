package harness

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	PayloadType          = "harness.pipeline.finished"
	SuccessOutputChannel = "success"
	FailedOutputChannel  = "failed"
	InitialPollInterval  = 30 * time.Second
	PollInterval         = 1 * time.Minute
)

// Terminal pipeline execution statuses in Harness.
var terminalStatuses = map[string]bool{
	"Success":              true,
	"Failed":               true,
	"Errored":              true,
	"Aborted":              true,
	"Expired":              true,
	"AbortedByFreeze":      true,
	"ApprovalRejected":     true,
	"InputWaitingAborted":  true,
}

var successStatuses = map[string]bool{
	"Success":       true,
	"IgnoreFailed":  true,
}

type RunPipeline struct{}

type RunPipelineConfiguration struct {
	OrgIdentifier      string `json:"orgIdentifier" mapstructure:"orgIdentifier"`
	ProjectIdentifier  string `json:"projectIdentifier" mapstructure:"projectIdentifier"`
	PipelineIdentifier string `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Module             string `json:"module" mapstructure:"module"`
}

type RunPipelineExecutionMetadata struct {
	Pipeline *RunPipelineMetadata `json:"pipeline" mapstructure:"pipeline"`
}

type RunPipelineMetadata struct {
	PlanExecutionID string `json:"planExecutionId"`
	Status          string `json:"status"`
	Name            string `json:"name"`
	ExecutionURL    string `json:"executionUrl"`
}

func (r *RunPipeline) Name() string {
	return "harness.runPipeline"
}

func (r *RunPipeline) Label() string {
	return "Run Pipeline"
}

func (r *RunPipeline) Description() string {
	return "Start a Harness pipeline and wait for it to complete"
}

func (r *RunPipeline) Documentation() string {
	return `The Run Pipeline component triggers a Harness pipeline execution and waits for it to complete.

## Use Cases

- **Deploy on merge**: Trigger a production deploy pipeline after a successful CI build
- **Scheduled pipelines**: Run pipelines on a schedule or in response to external events
- **Pipeline chaining**: Execute one pipeline when another finishes
- **Cross-project orchestration**: Trigger pipelines across different Harness projects

## How It Works

1. Triggers a pipeline execution via the Harness API
2. Monitors the pipeline status via polling
3. Routes execution based on pipeline outcome:
   - **Success channel**: Pipeline completed successfully
   - **Failed channel**: Pipeline failed, was aborted, or errored

## Configuration

- **Organization**: The Harness organization identifier
- **Project**: The Harness project identifier
- **Pipeline**: The pipeline identifier to execute
- **Module**: Optional module type (CI, CD, etc.)

## Output Channels

- **Success**: Emitted when the pipeline completes successfully
- **Failed**: Emitted when the pipeline fails, errors, or is aborted`
}

func (r *RunPipeline) Icon() string {
	return "workflow"
}

func (r *RunPipeline) Color() string {
	return "blue"
}

func (r *RunPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  SuccessOutputChannel,
			Label: "Success",
		},
		{
			Name:  FailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (r *RunPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orgIdentifier",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Harness organization identifier",
			Placeholder: "e.g. default",
		},
		{
			Name:        "projectIdentifier",
			Label:       "Project",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Harness project identifier",
			Placeholder: "e.g. my_project",
		},
		{
			Name:        "pipelineIdentifier",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Harness pipeline identifier",
			Placeholder: "e.g. my_pipeline",
		},
		{
			Name:        "module",
			Label:       "Module",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Module type for the pipeline",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "CI", Value: "CI"},
						{Label: "CD", Value: "CD"},
					},
				},
			},
		},
	}
}

func (r *RunPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunPipeline) Setup(ctx core.SetupContext) error {
	var config RunPipelineConfiguration
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.OrgIdentifier == "" {
		return fmt.Errorf("organization is required")
	}

	if config.ProjectIdentifier == "" {
		return fmt.Errorf("project is required")
	}

	if config.PipelineIdentifier == "" {
		return fmt.Errorf("pipeline is required")
	}

	return nil
}

func (r *RunPipeline) Execute(ctx core.ExecutionContext) error {
	var config RunPipelineConfiguration
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	response, err := client.ExecutePipeline(
		config.OrgIdentifier,
		config.ProjectIdentifier,
		config.PipelineIdentifier,
		config.Module,
	)
	if err != nil {
		return fmt.Errorf("error executing pipeline: %v", err)
	}

	if response.Data == nil || response.Data.PlanExecution == nil {
		return fmt.Errorf("unexpected response: no plan execution data")
	}

	planExecutionID := response.Data.PlanExecution.UUID
	executionURL := fmt.Sprintf(
		"https://app.harness.io/ng/account/%s/module/%s/orgs/%s/projects/%s/pipelines/%s/executions/%s/pipeline",
		client.AccountID,
		config.Module,
		config.OrgIdentifier,
		config.ProjectIdentifier,
		config.PipelineIdentifier,
		planExecutionID,
	)

	ctx.Logger.Infof("Pipeline execution started - planExecutionId=%s", planExecutionID)

	ctx.Metadata.Set(RunPipelineExecutionMetadata{
		Pipeline: &RunPipelineMetadata{
			PlanExecutionID: planExecutionID,
			Status:          response.Data.PlanExecution.Status,
			ExecutionURL:    executionURL,
		},
	})

	err = ctx.ExecutionState.SetKV("planExecutionId", planExecutionID)
	if err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, InitialPollInterval)
}

func (r *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (r *RunPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (r *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return r.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (r *RunPipeline) poll(ctx core.ActionContext) error {
	var config RunPipelineConfiguration
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata RunPipelineExecutionMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Pipeline == nil {
		return fmt.Errorf("pipeline metadata not found")
	}

	// If already in a terminal state, skip.
	if terminalStatuses[metadata.Pipeline.Status] {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	execution, err := client.GetPipelineExecution(
		config.OrgIdentifier,
		config.ProjectIdentifier,
		metadata.Pipeline.PlanExecutionID,
	)
	if err != nil {
		return err
	}

	if execution.Data == nil || execution.Data.PipelineExecution == nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	status := execution.Data.PipelineExecution.Status

	// If not in a terminal state, poll again.
	if !terminalStatuses[status] {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	metadata.Pipeline.Status = status
	metadata.Pipeline.Name = execution.Data.PipelineExecution.Name
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pipelineIdentifier": config.PipelineIdentifier,
		"planExecutionId":    metadata.Pipeline.PlanExecutionID,
		"status":             status,
		"name":               execution.Data.PipelineExecution.Name,
		"executionUrl":       metadata.Pipeline.ExecutionURL,
		"startTs":            execution.Data.PipelineExecution.StartTs,
		"endTs":              execution.Data.PipelineExecution.EndTs,
	}

	if successStatuses[status] {
		return ctx.ExecutionState.Emit(SuccessOutputChannel, PayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
}

func (r *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
