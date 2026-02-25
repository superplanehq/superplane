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

type GetPipelineExecution struct{}

type GetPipelineExecutionSpec struct {
	Region      string `json:"region" mapstructure:"region"`
	Pipeline    string `json:"pipeline" mapstructure:"pipeline"`
	ExecutionID string `json:"executionId" mapstructure:"executionId"`
}

func (c *GetPipelineExecution) Name() string {
	return "aws.codepipeline.getPipelineExecution"
}

func (c *GetPipelineExecution) Label() string {
	return "CodePipeline • Get Pipeline Execution"
}

func (c *GetPipelineExecution) Description() string {
	return "Retrieve the status and details of an AWS CodePipeline execution"
}

func (c *GetPipelineExecution) Documentation() string {
	return `The Get Pipeline Execution component retrieves the details of a specific AWS CodePipeline execution.

## Use Cases

- **Execution inspection**: Fetch the status, trigger, and artifact revisions of a pipeline run
- **Post-deploy checks**: After a RunPipeline component, fetch details of that execution for logging
- **Workflow branching**: Route workflow based on execution status or trigger type
- **Audit and compliance**: Retrieve execution details for auditing purposes

## Configuration

- **Region**: AWS region where the pipeline exists
- **Pipeline**: Pipeline name
- **Execution ID**: The ID of the specific execution to retrieve

## Output

Emits the full pipeline execution details including:
- Execution ID, status, and status summary
- Pipeline name and version
- Trigger type and detail
- Artifact revisions (source code revisions involved)
- Execution mode and type`
}

func (c *GetPipelineExecution) Icon() string {
	return "aws"
}

func (c *GetPipelineExecution) Color() string {
	return "orange"
}

func (c *GetPipelineExecution) Configuration() []configuration.Field {
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
			Description: "CodePipeline pipeline to query",
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
			Name:        "executionId",
			Label:       "Execution ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Pipeline execution ID to retrieve (supports expressions)",
		},
	}
}

func (c *GetPipelineExecution) Setup(ctx core.SetupContext) error {
	spec := GetPipelineExecutionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Region) == "" {
		return fmt.Errorf("region is required")
	}

	if strings.TrimSpace(spec.Pipeline) == "" {
		return fmt.Errorf("pipeline is required")
	}

	if strings.TrimSpace(spec.ExecutionID) == "" {
		return fmt.Errorf("execution ID is required")
	}

	return nil
}

func (c *GetPipelineExecution) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineExecutionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, strings.TrimSpace(spec.Region))

	response, err := client.GetPipelineExecutionDetails(strings.TrimSpace(spec.Pipeline), strings.TrimSpace(spec.ExecutionID))
	if err != nil {
		return fmt.Errorf("failed to get pipeline execution: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codepipeline.pipeline.execution",
		[]any{response.PipelineExecution},
	)
}

func (c *GetPipelineExecution) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPipelineExecution) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPipelineExecution) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPipelineExecution) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPipelineExecution) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetPipelineExecution) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPipelineExecution) Cleanup(ctx core.SetupContext) error {
	return nil
}
