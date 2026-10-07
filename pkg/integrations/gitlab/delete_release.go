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

//go:embed example_output_delete_release.json
var exampleOutputDeleteRelease []byte

type DeleteRelease struct{}

type DeleteReleaseConfiguration struct {
	Project         string `mapstructure:"project"`
	ReleaseStrategy string `mapstructure:"releaseStrategy"`
	TagName         string `mapstructure:"tagName"`
	DeleteTag       bool   `mapstructure:"deleteTag"`
}

func (c *DeleteRelease) Name() string {
	return "gitlab.deleteRelease"
}

func (c *DeleteRelease) Label() string {
	return "Delete Release"
}

func (c *DeleteRelease) Description() string {
	return "Delete a release from a GitLab project"
}

func (c *DeleteRelease) Documentation() string {
	return `The Delete Release component removes a release from a GitLab project.

## Use Cases

- **Cleanup**: Remove draft or test releases created in error
- **Rollback**: Delete a release that shouldn't have been published
- **Automated maintenance**: Remove releases as part of a cleanup workflow

## Configuration

- **Project** (required): The GitLab project containing the release
- **Release Strategy** (required): Identify the release by a specific tag, or use the project's latest release
- **Tag Name** (required if strategy is "Specific tag"): The tag of the release to delete
- **Also delete Git tag** (optional): GitLab does not delete the underlying tag when a release is deleted, so enable this to remove it too

## Output

Returns the deleted release object, plus a "tag_deleted" field that is true only if the Git tag was also requested and successfully removed.`
}

func (c *DeleteRelease) Icon() string {
	return "gitlab"
}

func (c *DeleteRelease) Color() string {
	return "orange"
}

func (c *DeleteRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteRelease) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputDeleteRelease, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *DeleteRelease) Configuration() []configuration.Field {
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
			Label:    "Release to Delete",
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
			Description: "How to identify which release to delete",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "v1.0.0",
			Description: "The tag of the release to delete",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "releaseStrategy", Values: []string{ReleaseStrategySpecific}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "releaseStrategy", Values: []string{ReleaseStrategySpecific}},
			},
		},
		{
			Name:        "deleteTag",
			Label:       "Also delete Git tag",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "GitLab does not delete the underlying tag when a release is deleted; enable this to also remove it",
		},
	}
}

func (c *DeleteRelease) Setup(ctx core.SetupContext) error {
	var config DeleteReleaseConfiguration
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

func (c *DeleteRelease) Execute(ctx core.ExecutionContext) error {
	var config DeleteReleaseConfiguration
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

	release, err := client.DeleteRelease(context.Background(), config.Project, tagName)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	output, err := releaseWithTagDeletionStatus(release)
	if err != nil {
		return fmt.Errorf("failed to encode release: %w", err)
	}

	if config.DeleteTag {
		if err := client.DeleteTag(context.Background(), config.Project, tagName); err != nil {
			ctx.Logger.Warnf("Release deleted successfully, but failed to delete tag %s: %v", tagName, err)
		} else {
			output["tag_deleted"] = true
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.release",
		[]any{output},
	)
}

// releaseWithTagDeletionStatus flattens a release into a map with tag_deleted defaulted to false, so the field can be added without dropping any of GitLab's own release fields.
func releaseWithTagDeletionStatus(release *Release) (map[string]any, error) {
	body, err := json.Marshal(release)
	if err != nil {
		return nil, err
	}

	var output map[string]any
	if err := json.Unmarshal(body, &output); err != nil {
		return nil, err
	}

	output["tag_deleted"] = false
	return output, nil
}

func (c *DeleteRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *DeleteRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteRelease) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteRelease) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
