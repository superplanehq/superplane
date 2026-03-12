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
	InputMode   string `json:"inputMode" mapstructure:"inputMode"`
	ResourceURL string `json:"resourceUrl" mapstructure:"resourceUrl"`
	Location    string `json:"location" mapstructure:"location"`
	Repository  string `json:"repository" mapstructure:"repository"`
	Package     string `json:"package" mapstructure:"package"`
	Version     string `json:"version" mapstructure:"version"`
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

Provide either a **Resource URL** or the four fields below:

- **Resource URL**: Full resource URL of the image (e.g. ` + "`https://us-central1-docker.pkg.dev/project/repo/image@sha256:abc`" + `). Use this to pass a digest directly from an upstream event such as On Artifact Push.
- **Location**: The GCP region where the repository is located.
- **Repository**: The Artifact Registry repository containing the artifact.
- **Package**: The package (image, library, etc.) within the repository.
- **Version**: The version or tag to retrieve.

## Output

The full Version resource, including ` + "`name`" + `, ` + "`createTime`" + `, ` + "`updateTime`" + `, ` + "`description`" + `, ` + "`relatedTags`" + `, and ` + "`metadata`" + `.

## Supported Formats

Artifact Registry supports all package formats when using **Select from Registry** mode.
**Resource URL** mode is intended for container image URLs (for example from On Artifact Push events).`
}

func (c *GetArtifact) Icon() string  { return "gcp" }
func (c *GetArtifact) Color() string { return "gray" }

func (c *GetArtifact) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetArtifact) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "inputMode",
			Label:    "Input Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "url",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Resource URL", Value: "url"},
						{Label: "Select from Registry", Value: "select"},
					},
				},
			},
		},
		{
			Name:        "resourceUrl",
			Label:       "Resource URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Full resource URL of the artifact (e.g. from an On Artifact Push event).",
			Placeholder: "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "inputMode", Values: []string{"url"}},
			},
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the Artifact Registry region.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "inputMode", Values: []string{"select"}},
			},
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
			Required:    false,
			Description: "Select the Artifact Registry repository.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "inputMode", Values: []string{"select"}},
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
			Required:    false,
			Description: "Select the package (image, library, etc.) to retrieve.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "inputMode", Values: []string{"select"}},
				{Field: "location", Values: []string{"*"}},
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
			Required:    false,
			Description: "Select the version or tag to retrieve.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "inputMode", Values: []string{"select"}},
				{Field: "location", Values: []string{"*"}},
				{Field: "repository", Values: []string{"*"}},
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
	config.ResourceURL = sanitizeConfigValue(config.ResourceURL)
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
	if config.InputMode == "url" || config.InputMode == "" {
		if config.ResourceURL != "" && !strings.Contains(config.ResourceURL, "{{") {
			_, _, _, _, err := parseArtifactResourceURL(config.ResourceURL)
			return err
		}
		return nil
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

	useURLMode := config.InputMode == "url" || config.InputMode == ""
	location, repository, pkg, version := config.Location, config.Repository, config.Package, config.Version
	if useURLMode {
		if config.ResourceURL == "" {
			return ctx.ExecutionState.Fail("error", "resourceUrl is required in url mode")
		}

		location, repository, pkg, version, err = parseArtifactResourceURL(config.ResourceURL)
		if err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid resourceUrl: %v", err))
		}
	} else {
		if location == "" {
			return ctx.ExecutionState.Fail("error", "location is required")
		}
		if repository == "" {
			return ctx.ExecutionState.Fail("error", "repository is required")
		}
		if pkg == "" {
			return ctx.ExecutionState.Fail("error", "package is required")
		}
		if version == "" {
			return ctx.ExecutionState.Fail("error", "version is required")
		}
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	packageName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s", projectID, location, repository, pkg)
	reqURL := getVersionURL(packageName, version)

	responseBody, err := client.GetURL(context.Background(), reqURL)
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
func (c *GetArtifact) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *GetArtifact) Cancel(_ core.ExecutionContext) error { return nil }
func (c *GetArtifact) Cleanup(_ core.SetupContext) error    { return nil }
func (c *GetArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
