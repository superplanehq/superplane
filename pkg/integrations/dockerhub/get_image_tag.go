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

type GetImageTag struct{}

type GetImageTagConfiguration struct {
	Namespace  string `json:"namespace" mapstructure:"namespace"`
	Repository string `json:"repository" mapstructure:"repository"`
	Tag        string `json:"tag" mapstructure:"tag"`
}

func (c *GetImageTag) Name() string {
	return "dockerhub.getImageTag"
}

func (c *GetImageTag) Label() string {
	return "Get Image Tag"
}

func (c *GetImageTag) Description() string {
	return "Get metadata for a DockerHub image tag"
}

func (c *GetImageTag) Documentation() string {
	return `The Get Image Tag component retrieves metadata for a DockerHub image tag.

## Use Cases

- **Release automation**: Fetch tag metadata for deployments
- **Audit trails**: Resolve tag details for traceability
- **Insights**: Inspect image sizes, digests, and last pushed times

## Configuration

- **Repository**: DockerHub repository name, in the format of ` + "`namespace/name`" + `
- **Tag**: Image tag to retrieve (for example: ` + "`latest`" + ` or ` + "`v1.2.3`" + `)
`
}

func (c *GetImageTag) Icon() string {
	return "docker"
}

func (c *GetImageTag) Color() string {
	return "gray"
}

func (c *GetImageTag) ExampleOutput() map[string]any {
	return getImageTagExampleOutput()
}

func (c *GetImageTag) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetImageTag) Configuration() []configuration.Field {
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
			Placeholder: "latest",
		},
	}
}

func (c *GetImageTag) Setup(ctx core.SetupContext) error {
	var config GetImageTagConfiguration
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

	return nil
}

func (c *GetImageTag) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetImageTag) Execute(ctx core.ExecutionContext) error {
	var config GetImageTagConfiguration
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

	tagResponse, err := client.GetRepositoryTag(namespace, repositoryName, tag)
	if err != nil {
		return fmt.Errorf("failed to fetch image tag: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dockerhub.tag",
		[]any{tagResponse},
	)
}

func (c *GetImageTag) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetImageTag) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetImageTag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetImageTag) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetImageTag) Cleanup(ctx core.SetupContext) error {
	return nil
}
