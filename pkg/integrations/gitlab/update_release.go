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

//go:embed example_output_update_release.json
var exampleOutputUpdateRelease []byte

type UpdateRelease struct{}

type UpdateReleaseConfiguration struct {
	Project         string   `mapstructure:"project"`
	ReleaseStrategy string   `mapstructure:"releaseStrategy"`
	TagName         string   `mapstructure:"tagName"`
	Name            string   `mapstructure:"name"`
	Description     string   `mapstructure:"description"`
	Milestones      []string `mapstructure:"milestones"`
	ReleasedAt      string   `mapstructure:"releasedAt"`
}

// updateReleaseToggles tracks which fields were explicitly toggled on, since a toggled-on-but-empty value must still be sent (see update_issue.go).
type updateReleaseToggles struct {
	Name        bool
	Description bool
	Milestones  bool
	ReleasedAt  bool
}

func newUpdateReleaseToggles(raw map[string]any) updateReleaseToggles {
	enabled := func(field string) bool {
		v, ok := raw[field]
		return ok && v != nil
	}
	return updateReleaseToggles{
		Name:        enabled("name"),
		Description: enabled("description"),
		Milestones:  enabled("milestones"),
		ReleasedAt:  enabled("releasedAt"),
	}
}

func (t updateReleaseToggles) hasUpdates() bool {
	return t.Name || t.Description || t.Milestones || t.ReleasedAt
}

func (c *UpdateRelease) Name() string {
	return "gitlab.updateRelease"
}

func (c *UpdateRelease) Label() string {
	return "Update Release"
}

func (c *UpdateRelease) Description() string {
	return "Update an existing release in a GitLab project"
}

func (c *UpdateRelease) Documentation() string {
	return `The Update Release component modifies an existing GitLab release: its name, description, milestones, or release date.

## Use Cases

- **Release notes updates**: Fill in release notes after the release has been cut
- **Milestone linking**: Associate a release with milestones once they're finalized
- **Rescheduling**: Move an upcoming release's date

## Configuration

- **Project** (required): The GitLab project containing the release
- **Release Strategy** (required): Identify the release by a specific tag, or use the project's latest release
- **Tag Name** (required if strategy is "Specific tag"): The tag of the release to update
- **Release Name** (toggle): New release title
- **Description** (toggle): New release notes, in Markdown
- **Milestones** (toggle): Milestones to associate with the release, replacing any existing ones
- **Released At** (toggle): New release date

Each field besides Project, Release Strategy and Tag Name is toggled on individually, so only the fields you enable are sent in the update. At least one must be enabled. Enabling Milestones with nothing selected clears them. Released At is the exception: GitLab requires a valid date, so it must have a value when enabled.

## Output

Returns the updated release object.`
}

func (c *UpdateRelease) Icon() string {
	return "gitlab"
}

func (c *UpdateRelease) Color() string {
	return "orange"
}

func (c *UpdateRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateRelease) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputUpdateRelease, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *UpdateRelease) Configuration() []configuration.Field {
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
			Label:    "Release Strategy",
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
			Description: "How to identify which release to update",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "v1.0.0",
			Description: "The tag of the release to update",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "releaseStrategy", Values: []string{ReleaseStrategySpecific}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "releaseStrategy", Values: []string{ReleaseStrategySpecific}},
			},
		},
		{
			Name:      "name",
			Label:     "Release Name",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "description",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:        "milestones",
			Label:       "Milestones",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Milestones to associate. Replaces the existing milestones; selecting none clears them.",
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
			Name:      "releasedAt",
			Label:     "Released At",
			Type:      configuration.FieldTypeDateTime,
			Required:  false,
			Togglable: true,
		},
	}
}

func (c *UpdateRelease) Setup(ctx core.SetupContext) error {
	var config UpdateReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if err := validateUpdateRelease(ctx.Configuration, config); err != nil {
		return err
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *UpdateRelease) Execute(ctx core.ExecutionContext) error {
	var config UpdateReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateUpdateRelease(ctx.Configuration, config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	tagName, err := resolveReleaseTagName(client, config.Project, config.ReleaseStrategy, config.TagName)
	if err != nil {
		return err
	}

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newUpdateReleaseToggles(raw)
	req := &UpdateReleaseRequest{}

	if toggles.Name {
		req.Name = &config.Name
	}

	if toggles.Description {
		req.Description = &config.Description
	}

	if toggles.Milestones {
		milestones := config.Milestones
		if milestones == nil {
			milestones = []string{}
		}
		req.Milestones = &milestones
	}

	if toggles.ReleasedAt {
		req.ReleasedAt = &config.ReleasedAt
	}

	release, err := client.UpdateRelease(context.Background(), config.Project, tagName, req)
	if err != nil {
		return fmt.Errorf("failed to update release: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.release",
		[]any{release},
	)
}

func validateUpdateRelease(rawConfig any, config UpdateReleaseConfiguration) error {
	if config.ReleaseStrategy == ReleaseStrategySpecific && config.TagName == "" {
		return errors.New("tag name is required")
	}

	raw, _ := rawConfig.(map[string]any)
	toggles := newUpdateReleaseToggles(raw)
	if !toggles.hasUpdates() {
		return errors.New("at least one field must be enabled to update")
	}

	if toggles.ReleasedAt && config.ReleasedAt == "" {
		return errors.New("released at cannot be empty")
	}

	return nil
}

func (c *UpdateRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdateRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateRelease) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateRelease) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
