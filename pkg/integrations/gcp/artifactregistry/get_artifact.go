package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	getArtifactPayloadType   = "gcp.artifactregistry.version"
	getArtifactOutputChannel = "default"
)

type GetArtifact struct{}

type GetArtifactConfiguration struct {
	Location   string `json:"location" mapstructure:"location"`
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
	Version    string `json:"version" mapstructure:"version"`
}

func (c *GetArtifact) Name() string {
	return "gcp.artifactregistry.getArtifact"
}

func (c *GetArtifact) Label() string {
	return "Artifact Registry • Get Artifact"
}

func (c *GetArtifact) Description() string {
	return "Retrieve artifact version details from GCP Artifact Registry"
}

func (c *GetArtifact) Documentation() string {
	return `Retrieves the details of a specific artifact version from Google Artifact Registry.

## Configuration

- **Location** (required): The GCP region where the repository is located.
- **Repository** (required): The Artifact Registry repository containing the artifact.
- **Package** (required): The package (image, library, etc.) within the repository.
- **Version** (required): The version or tag to retrieve.

## Output

The full Version resource, including ` + "`name`" + `, ` + "`createTime`" + `, ` + "`updateTime`" + `, ` + "`description`" + `, ` + "`relatedTags`" + `, and ` + "`metadata`" + `.

## Supported Formats

Artifact Registry supports all package formats: Docker, Maven, npm, PyPI, Go, APT, YUM, Helm, and more.`
}

func (c *GetArtifact) Icon() string  { return "gcp" }
func (c *GetArtifact) Color() string { return "gray" }

func (c *GetArtifact) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetArtifact) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Artifact Registry region.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeLocation,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Artifact Registry repository.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "location", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRepository,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
		},
		{
			Name:        "package",
			Label:       "Package",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the package (image, library, etc.) to retrieve.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
				{Field: "repository", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "repository", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePackage,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "repository"}},
					},
				},
			},
		},
		{
			Name:        "version",
			Label:       "Version",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the version or tag to retrieve.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
				{Field: "repository", Values: []string{"*"}},
				{Field: "package", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "package", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeVersion,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "repository"}},
						{Name: "package", ValueFrom: &configuration.ParameterValueFrom{Field: "package"}},
					},
				},
			},
		},
	}
}

func decodeGetArtifactConfiguration(raw any) (GetArtifactConfiguration, error) {
	var config GetArtifactConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return GetArtifactConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Location = strings.TrimSpace(config.Location)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Package = strings.TrimSpace(config.Package)
	config.Version = strings.TrimSpace(config.Version)
	return config, nil
}

func (c *GetArtifact) Setup(ctx core.SetupContext) error {
	config, err := decodeGetArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Location == "" {
		return fmt.Errorf("location is required")
	}
	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if config.Package == "" {
		return fmt.Errorf("package is required")
	}
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

func (c *GetArtifact) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	packageName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s", projectID, config.Location, config.Repository, config.Package)
	url := getVersionURL(packageName, config.Version)

	responseBody, err := client.GetURL(context.Background(), url)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get artifact version: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse response: %v", err))
	}

	return ctx.ExecutionState.Emit(getArtifactOutputChannel, getArtifactPayloadType, []any{result})
}

func (c *GetArtifact) Actions() []core.Action                  { return nil }
func (c *GetArtifact) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetArtifact) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *GetArtifact) Cancel(_ core.ExecutionContext) error { return nil }
func (c *GetArtifact) Cleanup(_ core.SetupContext) error    { return nil }
func (c *GetArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
