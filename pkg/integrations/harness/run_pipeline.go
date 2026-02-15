package harness

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	RunPipelinePayloadType   = "harness.pipeline.finished"
	RunPipelinePollAction    = "poll"
	RunPipelineSuccess       = "success"
	RunPipelineFailed        = "failed"
	RunPipelinePollInterval  = 1 * time.Minute
	RunPipelineMaxPollErrors = 5
)

type RunPipeline struct{}

type RunPipelineSpec struct {
	PipelineIdentifier string   `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Ref                string   `json:"ref" mapstructure:"ref"`
	InputSetReferences []string `json:"inputSetReferences" mapstructure:"inputSetReferences"`
	InputYAML          string   `json:"inputYAML" mapstructure:"inputYAML"`
}

type RunPipelineExecutionMetadata struct {
	ExecutionID        string `json:"executionId" mapstructure:"executionId"`
	PipelineIdentifier string `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Status             string `json:"status" mapstructure:"status"`
	PlanExecutionURL   string `json:"planExecutionUrl,omitempty" mapstructure:"planExecutionUrl"`
	StartedAt          string `json:"startedAt,omitempty" mapstructure:"startedAt"`
	EndedAt            string `json:"endedAt,omitempty" mapstructure:"endedAt"`
	PollErrorCount     int    `json:"pollErrorCount,omitempty" mapstructure:"pollErrorCount"`
}

func (r *RunPipeline) Name() string {
	return "harness.runPipeline"
}

func (r *RunPipeline) Label() string {
	return "Run Pipeline"
}

func (r *RunPipeline) Description() string {
	return "Run a Harness pipeline and wait for completion"
}

func (r *RunPipeline) Documentation() string {
	return `The Run Pipeline component starts a Harness pipeline execution and waits for it to finish.

## Use Cases

- **CI/CD orchestration**: Trigger deploy pipelines from workflow events
- **Approval-based releases**: Run release pipelines after manual approvals
- **Scheduled automation**: Kick off recurring maintenance or validation pipelines

## How It Works

1. Starts a Harness pipeline execution
2. Stores the execution ID in node execution state
3. Polls execution summary until a terminal status is reached
4. Routes output to:
   - **Success** when execution succeeds
   - **Failed** when execution fails, aborts, or expires

## Configuration

- **Pipeline**: Harness pipeline identifier
- **Ref**: Optional git ref (` + "`refs/heads/main`" + ` or ` + "`refs/tags/v1.2.3`" + `)
- **Input Set References**: Optional input set identifiers
- **Runtime Input YAML**: Optional YAML override for runtime inputs`
}

func (r *RunPipeline) Icon() string {
	return "workflow"
}

func (r *RunPipeline) Color() string {
	return "gray"
}

func (r *RunPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: RunPipelineSuccess, Label: "Success"},
		{Name: RunPipelineFailed, Label: "Failed"},
	}
}

func (r *RunPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pipelineIdentifier",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Harness pipeline identifier",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePipeline,
				},
			},
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeGitRef,
			Required:    false,
			Default:     "refs/heads/main",
			Description: "Optional branch or tag ref",
		},
		{
			Name:        "inputSetReferences",
			Label:       "Input Set References",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional Harness input set references",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Input Set",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "inputYAML",
			Label:       "Runtime Input YAML",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional runtime input YAML override",
		},
	}
}

func (r *RunPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunPipeline) Setup(ctx core.SetupContext) error {
	_, err := decodeRunPipelineSpec(ctx.Configuration)
	return err
}

func (r *RunPipeline) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	response, err := client.RunPipeline(RunPipelineRequest{
		PipelineIdentifier: spec.PipelineIdentifier,
		Ref:                spec.Ref,
		InputSetRefs:       spec.InputSetReferences,
		InputYAML:          spec.InputYAML,
	})
	if err != nil {
		return err
	}

	metadata := RunPipelineExecutionMetadata{
		ExecutionID:        response.ExecutionID,
		PipelineIdentifier: spec.PipelineIdentifier,
		Status:             "running",
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV("execution", response.ExecutionID); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
}

func (r *RunPipeline) Actions() []core.Action {
	return []core.Action{{Name: RunPipelinePollAction, UserAccessible: false}}
}

func (r *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case RunPipelinePollAction:
		return r.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (r *RunPipeline) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if strings.TrimSpace(metadata.ExecutionID) == "" {
		return fmt.Errorf("execution ID is missing from metadata")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	summary, err := client.GetExecutionSummary(metadata.ExecutionID)
	if err != nil {
		metadata.PollErrorCount++
		metadata.Status = "error"

		if setErr := ctx.Metadata.Set(metadata); setErr != nil {
			return setErr
		}

		if metadata.PollErrorCount >= RunPipelineMaxPollErrors {
			payload := map[string]any{
				"executionId":        metadata.ExecutionID,
				"pipelineIdentifier": metadata.PipelineIdentifier,
				"status":             canonicalStatus(metadata.Status),
				"planExecutionUrl":   metadata.PlanExecutionURL,
				"startedAt":          metadata.StartedAt,
				"endedAt":            metadata.EndedAt,
				"error":              err.Error(),
			}
			return ctx.ExecutionState.Emit(RunPipelineFailed, RunPipelinePayloadType, []any{payload})
		}

		return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
	}

	metadata.Status = summary.Status
	metadata.PlanExecutionURL = summary.PlanExecutionURL
	metadata.StartedAt = summary.StartedAt
	metadata.EndedAt = summary.EndedAt
	metadata.PollErrorCount = 0
	if metadata.PipelineIdentifier == "" {
		metadata.PipelineIdentifier = summary.PipelineIdentifier
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if !isTerminalStatus(summary.Status) {
		return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
	}

	payload := map[string]any{
		"executionId":        metadata.ExecutionID,
		"pipelineIdentifier": metadata.PipelineIdentifier,
		"status":             canonicalStatus(summary.Status),
		"planExecutionUrl":   metadata.PlanExecutionURL,
		"startedAt":          metadata.StartedAt,
		"endedAt":            metadata.EndedAt,
	}

	if isSuccessStatus(summary.Status) {
		return ctx.ExecutionState.Emit(RunPipelineSuccess, RunPipelinePayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(RunPipelineFailed, RunPipelinePayloadType, []any{payload})
}

func (r *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (r *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeRunPipelineSpec(value any) (RunPipelineSpec, error) {
	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(value, &spec); err != nil {
		return RunPipelineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.PipelineIdentifier = strings.TrimSpace(spec.PipelineIdentifier)
	if spec.PipelineIdentifier == "" {
		return RunPipelineSpec{}, fmt.Errorf("pipelineIdentifier is required")
	}

	spec.Ref = strings.TrimSpace(spec.Ref)
	spec.InputYAML = strings.TrimSpace(spec.InputYAML)

	filtered := make([]string, 0, len(spec.InputSetReferences))
	for _, ref := range spec.InputSetReferences {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	spec.InputSetReferences = filtered

	return spec, nil
}
