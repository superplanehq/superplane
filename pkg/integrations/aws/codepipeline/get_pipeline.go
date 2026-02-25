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

type GetPipeline struct{}

type GetPipelineSpec struct {
	Region   string `json:"region" mapstructure:"region"`
	Pipeline string `json:"pipeline" mapstructure:"pipeline"`
}

func (c *GetPipeline) Name() string {
	return "aws.codepipeline.getPipeline"
}

func (c *GetPipeline) Label() string {
	return "CodePipeline • Get Pipeline"
}

func (c *GetPipeline) Description() string {
	return "Retrieve the definition of an AWS CodePipeline pipeline"
}

func (c *GetPipeline) Documentation() string {
	return `The Get Pipeline component retrieves the full definition of an AWS CodePipeline pipeline.

## Use Cases

- **Pipeline inspection**: Fetch pipeline stages, actions, and configuration
- **Workflow branching**: Route workflow based on pipeline structure or version
- **Audit and compliance**: Retrieve pipeline definitions for auditing purposes

## Configuration

- **Region**: AWS region where the pipeline exists
- **Pipeline**: Pipeline name to retrieve

## Output

Emits the full pipeline definition including:
- Pipeline name, version, and role ARN
- All stages and their actions
- Pipeline metadata (ARN, creation date, last updated date)`
}

func (c *GetPipeline) Icon() string {
	return "aws"
}

func (c *GetPipeline) Color() string {
	return "orange"
}

func (c *GetPipeline) Configuration() []configuration.Field {
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
			Description: "CodePipeline pipeline to retrieve",
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
	}
}

func (c *GetPipeline) Setup(ctx core.SetupContext) error {
	spec := GetPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Region) == "" {
		return fmt.Errorf("region is required")
	}

	if strings.TrimSpace(spec.Pipeline) == "" {
		return fmt.Errorf("pipeline is required")
	}

	return nil
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	response, err := client.GetPipeline(spec.Pipeline)
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codepipeline.pipeline",
		[]any{response},
	)
}

func (c *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
