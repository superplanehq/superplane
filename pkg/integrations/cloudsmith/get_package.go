package cloudsmith

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetPackage struct{}

type GetPackageConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Identifier string `json:"identifier" mapstructure:"identifier"`
}

func (c *GetPackage) Name() string {
	return "cloudsmith.getPackage"
}

func (c *GetPackage) Label() string {
	return "Get Package"
}

func (c *GetPackage) Description() string {
	return "Fetch details of a Cloudsmith package"
}

func (c *GetPackage) Documentation() string {
	return `The Get Package component retrieves metadata for a Cloudsmith package.

## Use Cases

- **Release automation**: Fetch package details for downstream deployments
- **Audit trails**: Resolve package metadata for traceability
- **Insights**: Inspect package sizes, checksums, and status

## Configuration

- **Repository**: Cloudsmith repository in the format ` + "`namespace/repo`" + `
- **Identifier**: Package identifier or slug (for example: ` + "`Wklm1a2b`" + `)
`
}

func (c *GetPackage) Icon() string {
	return "package"
}

func (c *GetPackage) Color() string {
	return "gray"
}

func (c *GetPackage) ExampleOutput() map[string]any {
	return getPackageExampleOutput()
}

func (c *GetPackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPackage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRepository,
				},
			},
		},
		{
			Name:        "identifier",
			Label:       "Identifier",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "Wklm1a2b",
			Description: "Package identifier or slug",
		},
	}
}

func (c *GetPackage) Setup(ctx core.SetupContext) error {
	var config GetPackageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository := strings.TrimSpace(config.Repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}

	identifier := strings.TrimSpace(config.Identifier)
	if identifier == "" {
		return fmt.Errorf("identifier is required")
	}

	return nil
}

func (c *GetPackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPackage) Execute(ctx core.ExecutionContext) error {
	var config GetPackageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository := strings.TrimSpace(config.Repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}

	identifier := strings.TrimSpace(config.Identifier)
	if identifier == "" {
		return fmt.Errorf("identifier is required")
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in the format of namespace/repo")
	}

	namespace := strings.TrimSpace(parts[0])
	repoSlug := strings.TrimSpace(parts[1])

	if namespace == "" || repoSlug == "" {
		return fmt.Errorf("repository must be in the format of namespace/repo")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	pkg, err := client.GetPackage(namespace, repoSlug, identifier)
	if err != nil {
		return fmt.Errorf("failed to fetch package: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package",
		[]any{pkg},
	)
}

func (c *GetPackage) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPackage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPackage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetPackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPackage) Cleanup(ctx core.SetupContext) error {
	return nil
}
