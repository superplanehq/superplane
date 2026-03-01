package codepipeline

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type RetryStageExecution struct{}

const (
	RetryModeFailedActions = "FAILED_ACTIONS"
	RetryModeAllActions    = "ALL_ACTIONS"
)

type RetryStageExecutionSpec struct {
	Region            string `json:"region" mapstructure:"region"`
	Pipeline          string `json:"pipeline" mapstructure:"pipeline"`
	Stage             string `json:"stage" mapstructure:"stage"`
	PipelineExecution string `json:"pipelineExecution" mapstructure:"pipelineExecution"`
	RetryMode         string `json:"retryMode" mapstructure:"retryMode"`
}

func normalizeRetryStageExecutionSpec(spec *RetryStageExecutionSpec) {
	spec.Region = strings.TrimSpace(spec.Region)
	spec.Pipeline = strings.TrimSpace(spec.Pipeline)
	spec.Stage = strings.TrimSpace(spec.Stage)
	spec.PipelineExecution = strings.TrimSpace(spec.PipelineExecution)
	spec.RetryMode = strings.TrimSpace(spec.RetryMode)
}

func (c *RetryStageExecution) Name() string {
	return "aws.codepipeline.retryStageExecution"
}

func (c *RetryStageExecution) Label() string {
	return "CodePipeline • Retry Stage Execution"
}

func (c *RetryStageExecution) Description() string {
	return "Retry a failed stage in an existing AWS CodePipeline execution"
}

func (c *RetryStageExecution) Documentation() string {
	return `The Retry Stage Execution component retries a stage within an existing AWS CodePipeline execution.

## Use Cases

- **Recover failed deployments**: Retry only failed actions in a failed stage
- **Re-run full stage**: Retry all actions for a stage when needed
- **Workflow recovery**: Continue orchestration after a transient failure

## Configuration

- **Region**: AWS region where the pipeline exists
- **Pipeline**: Pipeline name
- **Stage**: Stage name to retry
- **Pipeline Execution**: Source execution to retry from
- **Retry Mode**: Choose between failed actions only or all actions

## Output

Emits retry result metadata including:
- Pipeline name and stage
- Selected retry mode
- Source execution ID
- New execution ID created by the retry`
}

func (c *RetryStageExecution) Icon() string {
	return "aws"
}

func (c *RetryStageExecution) Color() string {
	return "orange"
}

func (c *RetryStageExecution) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RetryStageExecution) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "pipeline",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "CodePipeline pipeline to retry",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codepipeline.pipeline",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "pipelineExecution",
			Label:       "Pipeline Execution",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Pipeline execution to retry",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codepipeline.pipelineExecution",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "pipeline",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "pipeline",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "pipeline",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "stage",
			Label:       "Stage",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Stage to retry",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codepipeline.stage",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "pipeline",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "pipeline",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "pipeline",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:     "retryMode",
			Label:    "Retry Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  RetryModeFailedActions,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Failed Actions", Value: RetryModeFailedActions},
						{Label: "All Actions", Value: RetryModeAllActions},
					},
				},
			},
		},
	}
}

func (c *RetryStageExecution) Setup(ctx core.SetupContext) error {
	spec := RetryStageExecutionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeRetryStageExecutionSpec(&spec)

	if spec.Region == "" {
		return fmt.Errorf("region is required")
	}
	if spec.Pipeline == "" {
		return fmt.Errorf("pipeline is required")
	}
	if spec.Stage == "" {
		return fmt.Errorf("stage is required")
	}
	if spec.PipelineExecution == "" {
		return fmt.Errorf("pipeline execution is required")
	}
	if spec.RetryMode == "" {
		return fmt.Errorf("retry mode is required")
	}
	if spec.RetryMode != RetryModeFailedActions && spec.RetryMode != RetryModeAllActions {
		return fmt.Errorf(
			"retry mode must be one of %s, %s",
			RetryModeFailedActions,
			RetryModeAllActions,
		)
	}

	return nil
}

func (c *RetryStageExecution) Execute(ctx core.ExecutionContext) error {
	spec := RetryStageExecutionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeRetryStageExecutionSpec(&spec)

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	response, err := client.RetryStageExecution(spec.Pipeline, spec.Stage, spec.PipelineExecution, spec.RetryMode)
	if err != nil {
		return fmt.Errorf("failed to retry stage execution: %w", err)
	}

	payload := map[string]any{
		"pipeline": map[string]any{
			"name":              spec.Pipeline,
			"stage":             spec.Stage,
			"retryMode":         spec.RetryMode,
			"sourceExecutionId": spec.PipelineExecution,
			"newExecutionId":    response.PipelineExecutionID,
		},
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.codepipeline.stage.retry", []any{payload})
}

func (c *RetryStageExecution) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RetryStageExecution) Actions() []core.Action {
	return []core.Action{}
}

func (c *RetryStageExecution) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RetryStageExecution) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RetryStageExecution) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RetryStageExecution) Cleanup(ctx core.SetupContext) error {
	return nil
}
