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

//go:embed example_output_create_release.json
var exampleOutputCreateRelease []byte

type CreateRelease struct{}

type CreateReleaseConfiguration struct {
	Project     string   `mapstructure:"project"`
	TagName     string   `mapstructure:"tagName"`
	Ref         string   `mapstructure:"ref"`
	Name        string   `mapstructure:"name"`
	Description string   `mapstructure:"description"`
	Milestones  []string `mapstructure:"milestones"`
	ReleasedAt  string   `mapstructure:"releasedAt"`
}

func (c *CreateRelease) Name() string {
	return "gitlab.createRelease"
}

func (c *CreateRelease) Label() string {
	return "Create Release"
}

func (c *CreateRelease) Description() string {
	return "Create a new release in a GitLab project"
}

func (c *CreateRelease) Documentation() string {
	return `The Create Release component creates a new release in a GitLab project, optionally creating the underlying tag at the same time.

## Use Cases

- **Automated releases**: Cut a release automatically after a pipeline succeeds
- **Tag and release together**: Create the git tag and its release notes in one step
- **Scheduled releases**: Set a future release date to publish an upcoming release

## Configuration

- **Project** (required): The GitLab project to release in
- **Tag Name** (required): The tag to create the release for (supports expressions)
- **Ref** (optional): Branch, tag, or commit SHA to create the tag from. Required if the tag doesn't already exist
- **Release Name** (optional): The release title
- **Description** (optional): Release notes, in Markdown
- **Milestones** (optional): Project milestones to associate with the release
- **Released At** (optional): Set a date in the future to schedule an upcoming release, or in the past to backdate it. Defaults to now

## Output

Returns the created release object.`
}

func (c *CreateRelease) Icon() string {
	return "gitlab"
}

func (c *CreateRelease) Color() string {
	return "orange"
}

func (c *CreateRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRelease) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateRelease, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *CreateRelease) Configuration() []configuration.Field {
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
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "v1.0.0",
			Description: "The tag to create the release for",
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "main or a commit SHA",
			Description: "Branch, tag, or commit to create the tag from. Required if the tag doesn't already exist",
		},
		{
			Name:        "name",
			Label:       "Release Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Release 1.0.0",
			Description: "The title of the release. Defaults to the tag name if left empty",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "## What's Changed\n\n...",
			Description: "Release notes, in Markdown",
		},
		{
			Name:     "milestones",
			Label:    "Milestones",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeMilestone,
					Multi:          true,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:        "releasedAt",
			Label:       "Released At",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Defaults to now. Set a future date to schedule an upcoming release",
		},
	}
}

func (c *CreateRelease) Setup(ctx core.SetupContext) error {
	var config CreateReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.TagName == "" {
		return errors.New("tag name is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *CreateRelease) Execute(ctx core.ExecutionContext) error {
	var config CreateReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.TagName == "" {
		return errors.New("tag name is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	req := &CreateReleaseRequest{
		TagName:     config.TagName,
		Ref:         config.Ref,
		Name:        config.Name,
		Description: config.Description,
		Milestones:  config.Milestones,
		ReleasedAt:  config.ReleasedAt,
	}

	release, err := client.CreateRelease(context.Background(), config.Project, req)
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.release",
		[]any{release},
	)
}

func (c *CreateRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateRelease) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateRelease) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
