package gitlab

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_latest_pipeline.json
var exampleOutputGetLatestPipeline []byte

type GetLatestPipeline struct{}

type GetLatestPipelineConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
	Ref     string `json:"ref" mapstructure:"ref"`
}

func (c *GetLatestPipeline) Name() string {
	return "gitlab.getLatestPipeline"
}

func (c *GetLatestPipeline) Label() string {
	return "Get Latest Pipeline"
}

func (c *GetLatestPipeline) Description() string {
	return "Get the latest GitLab pipeline for a project"
}

func (c *GetLatestPipeline) Documentation() string {
	return `The Get Latest Pipeline component retrieves the newest pipeline for a GitLab project.

## Configuration

- **Project** (required): The GitLab project to query
- **Ref** (optional): Branch or tag to scope the latest pipeline search`
}

func (c *GetLatestPipeline) Icon() string {
	return "gitlab"
}

func (c *GetLatestPipeline) Color() string {
	return "orange"
}

func (c *GetLatestPipeline) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetLatestPipeline, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetLatestPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetLatestPipeline) Configuration() []configuration.Field {
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
			Name:  "ref",
			Label: "Ref",
			Type:  configuration.FieldTypeGitRef,
		},
	}
}

func (c *GetLatestPipeline) Setup(ctx core.SetupContext) error {
	var config GetLatestPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project)
}

func (c *GetLatestPipeline) Execute(ctx core.ExecutionContext) error {
	var config GetLatestPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetLatestPipeline(config.Project, normalizePipelineRef(config.Ref))
	if err != nil {
		return fmt.Errorf("failed to get latest pipeline: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gitlab.pipeline", []any{pipeline})
}

func (c *GetLatestPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetLatestPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetLatestPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetLatestPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetLatestPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetLatestPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
