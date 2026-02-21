package gitlab

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_pipeline.json
var exampleOutputGetPipeline []byte

type GetPipeline struct{}

type GetPipelineConfiguration struct {
	Project  string `json:"project" mapstructure:"project"`
	Pipeline string `json:"pipeline" mapstructure:"pipeline"`
}

func (c *GetPipeline) Name() string {
	return "gitlab.getPipeline"
}

func (c *GetPipeline) Label() string {
	return "Get Pipeline"
}

func (c *GetPipeline) Description() string {
	return "Get a GitLab pipeline"
}

func (c *GetPipeline) Documentation() string {
	return `The Get Pipeline component retrieves details for a specific GitLab pipeline.

## Configuration

- **Project** (required): The GitLab project containing the pipeline
- **Pipeline** (required): Select a pipeline from the selected project

## Output

Returns pipeline data including status, ref, SHA, and pipeline URL.`
}

func (c *GetPipeline) Icon() string {
	return "gitlab"
}

func (c *GetPipeline) Color() string {
	return "orange"
}

func (c *GetPipeline) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetPipeline, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "pipeline",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. 1234567890",
			Description: "The ID of the pipeline to get",
		},
	}
}

func (c *GetPipeline) Setup(ctx core.SetupContext) error {
	var config GetPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Pipeline == "" {
		return fmt.Errorf("pipeline is required")
	}

	return ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project)
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) error {
	var config GetPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	p, err := strconv.ParseFloat(config.Pipeline, 64)
	if err != nil {
		return fmt.Errorf("pipeline ID must be a number: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(config.Project, int(p))
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gitlab.pipeline", []any{pipeline})
}

func (c *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
