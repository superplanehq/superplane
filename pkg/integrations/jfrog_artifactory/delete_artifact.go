package jfrogartifactory

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	DeleteArtifactPayloadType = "jfrogArtifactory.artifact.deleted"
)

type DeleteArtifact struct{}

type DeleteArtifactSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Path       string `json:"path" mapstructure:"path"`
}

type DeleteArtifactNodeMetadata struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

func (d *DeleteArtifact) Name() string {
	return "jfrogArtifactory.deleteArtifact"
}

func (d *DeleteArtifact) Label() string {
	return "Delete Artifact"
}

func (d *DeleteArtifact) Description() string {
	return "Delete an artifact from JFrog Artifactory"
}

func (d *DeleteArtifact) Documentation() string {
	return `The Delete Artifact component removes an artifact from a JFrog Artifactory repository.

## Use Cases

- **Cleanup pipelines**: Remove outdated or temporary artifacts after a release
- **Storage management**: Delete artifacts that are no longer needed
- **Automated housekeeping**: Trigger deletions based on workflow conditions

## Configuration

- **Repository**: Select the Artifactory repository containing the artifact
- **Path**: The path to the artifact within the repository (supports expressions)

## Output

Returns the repository and path of the deleted artifact.`
}

func (d *DeleteArtifact) Icon() string {
	return "jfrogArtifactory"
}

func (d *DeleteArtifact) Color() string {
	return "gray"
}

func (d *DeleteArtifact) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteArtifact) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "path",
			Label:       "Path",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Path to the artifact within the repository",
			Placeholder: "e.g. path/to/file.jar",
		},
	}
}

func (d *DeleteArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteArtifact) Setup(ctx core.SetupContext) error {
	spec := DeleteArtifactSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	return ctx.Metadata.Set(DeleteArtifactNodeMetadata{
		Repository: spec.Repository,
	})
}

func (d *DeleteArtifact) Execute(ctx core.ExecutionContext) error {
	spec := DeleteArtifactSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.DeleteArtifact(spec.Repository, spec.Path); err != nil {
		return fmt.Errorf("error deleting artifact: %v", err)
	}

	payload := map[string]any{
		"repo": spec.Repository,
		"path": spec.Path,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteArtifactPayloadType, []any{payload})
}

func (d *DeleteArtifact) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteArtifact) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (d *DeleteArtifact) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteArtifact) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteArtifact) Cleanup(ctx core.SetupContext) error {
	return nil
}
