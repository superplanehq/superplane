package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_release.json
var exampleOutputGetRelease []byte

type GetRelease struct{}

type GetReleaseConfiguration struct {
	Project         string `mapstructure:"project"`
	ReleaseStrategy string `mapstructure:"releaseStrategy"`
	TagName         string `mapstructure:"tagName"`
}

func (c *GetRelease) Name() string {
	return "gitlab.getRelease"
}

func (c *GetRelease) Label() string {
	return "Get Release"
}

func (c *GetRelease) Description() string {
	return "Get a release from a GitLab project"
}

func (c *GetRelease) Documentation() string {
	return `The Get Release component retrieves release information from a GitLab project.

## Use Cases

- **Release monitoring**: Fetch release details for use in downstream steps
- **Deployment pipelines**: Read release assets and metadata for deployment
- **Version tracking**: Check the project's latest published release

## Configuration

- **Project** (required): The GitLab project containing the release
- **Release Strategy** (required): Identify the release by a specific tag, or use the project's latest release
- **Tag Name** (required if strategy is "Specific tag"): The tag of the release to retrieve

## Output

Returns the release object.`
}

func (c *GetRelease) Icon() string {
	return "gitlab"
}

func (c *GetRelease) Color() string {
	return "orange"
}

func (c *GetRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRelease) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetRelease, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetRelease) Configuration() []configuration.Field {
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
			Name:     "releaseStrategy",
			Label:    "Release to Get",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  ReleaseStrategySpecific,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Specific tag", Value: ReleaseStrategySpecific},
						{Label: "Latest release", Value: ReleaseStrategyLatest},
					},
				},
			},
			Description: "How to identify which release to retrieve",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "v1.0.0",
			Description: "The tag of the release to retrieve",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "releaseStrategy", Values: []string{ReleaseStrategySpecific}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "releaseStrategy", Values: []string{ReleaseStrategySpecific}},
			},
		},
	}
}

func (c *GetRelease) Setup(ctx core.SetupContext) error {
	var config GetReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.ReleaseStrategy == ReleaseStrategySpecific && config.TagName == "" {
		return errors.New("tag name is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *GetRelease) Execute(ctx core.ExecutionContext) error {
	var config GetReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ReleaseStrategy == ReleaseStrategySpecific && config.TagName == "" {
		return errors.New("tag name is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	tagName, err := resolveReleaseTagName(client, config.Project, config.ReleaseStrategy, config.TagName)
	if err != nil {
		return err
	}

	release, err := client.GetRelease(context.Background(), config.Project, tagName)
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.release",
		[]any{release},
	)
}

func (c *GetRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetRelease) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetRelease) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
