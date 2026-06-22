package dockerhub

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteTag struct{}

type DeleteTagConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Tag        string `json:"tag" mapstructure:"tag"`
}

func (c *DeleteTag) Name() string {
	return "dockerhub.deleteTag"
}

func (c *DeleteTag) Label() string {
	return "Delete Tag"
}

func (c *DeleteTag) Description() string {
	return "Delete a tag from a DockerHub repository"
}

func (c *DeleteTag) Documentation() string {
	return `The Delete Tag component permanently removes a tag from a DockerHub repository.

## Use Cases

- **Cleanup pipelines**: Remove stale or temporary tags after a deployment succeeds
- **Release workflows**: Delete RC or beta tags once a release is promoted to stable
- **Policy enforcement**: Prune tags that violate naming conventions

## Configuration

- **Repository**: DockerHub repository name, in the format of ` + "`namespace/name`" + `
- **Tag**: Image tag to delete (for example: ` + "`v1.2.3-rc1`" + `)

> **Warning**: This action is irreversible. The tag cannot be recovered after deletion.
`
}

func (c *DeleteTag) Icon() string {
	return "docker"
}

func (c *DeleteTag) Color() string {
	return "gray"
}

func (c *DeleteTag) ExampleOutput() map[string]any {
	return deleteTagExampleOutput()
}

func (c *DeleteTag) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteTag) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "dockerhub.repository",
				},
			},
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "v1.2.3-rc1",
		},
	}
}

func (c *DeleteTag) Setup(ctx core.SetupContext) error {
	var config DeleteTagConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Repository) == "" {
		return fmt.Errorf("repository is required")
	}

	if strings.TrimSpace(config.Tag) == "" {
		return fmt.Errorf("tag is required")
	}

	return nil
}

func (c *DeleteTag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteTag) Execute(ctx core.ExecutionContext) error {
	var config DeleteTagConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository := strings.TrimSpace(config.Repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}

	tag := strings.TrimSpace(config.Tag)
	if tag == "" {
		return fmt.Errorf("tag is required")
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in the format of namespace/name")
	}

	namespace := strings.TrimSpace(parts[0])
	repositoryName := strings.TrimSpace(parts[1])
	if namespace == "" || repositoryName == "" {
		return fmt.Errorf("repository must be in the format of namespace/name")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.DeleteRepositoryTag(namespace, repositoryName, tag); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dockerhub.deletedTag",
		[]any{map[string]any{
			"namespace":  namespace,
			"repository": repositoryName,
			"tag":        tag,
		}},
	)
}

func (c *DeleteTag) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteTag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteTag) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteTag) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteTag) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
