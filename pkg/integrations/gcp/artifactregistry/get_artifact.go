package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	getArtifactPayloadType   = "gcp.artifactregistry.dockerImage"
	getArtifactOutputChannel = "default"
)

type GetArtifact struct{}

type GetArtifactConfiguration struct {
	Location   string `json:"location" mapstructure:"location"`
	Repository string `json:"repository" mapstructure:"repository"`
	Image      string `json:"image" mapstructure:"image"`
}

func (c *GetArtifact) Name() string {
	return "gcp.artifactregistry.getArtifact"
}

func (c *GetArtifact) Label() string {
	return "Artifact Registry • Get Artifact"
}

func (c *GetArtifact) Description() string {
	return "Retrieve a Docker image from Artifact Registry"
}

func (c *GetArtifact) Documentation() string {
	return `Retrieves the details of a Docker image stored in a Google Artifact Registry repository.

## Configuration

- **Location** (required): GCP region of the Artifact Registry repository (e.g. ` + "`us-central1`" + `).
- **Repository** (required): Name of the Artifact Registry repository.
- **Image** (required): Docker image name including the digest. Use the format ` + "`image-name@sha256:digest`" + ` or the short digest path returned by GCP (e.g. ` + "`sha256:abc123`" + `).

## Output

The full DockerImage resource, including ` + "`name`" + `, ` + "`uri`" + `, ` + "`tags`" + `, ` + "`imageSizeBytes`" + `, ` + "`uploadTime`" + `, ` + "`mediaType`" + `, ` + "`buildTime`" + `, and ` + "`updateTime`" + `.`
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
			Description: "GCP region of the Artifact Registry repository.",
			Placeholder: "Select a location",
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
			Description: "Artifact Registry repository name.",
			Placeholder: "Select a repository",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeRepository,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Docker image name with digest (e.g. my-image@sha256:abc123).",
			Placeholder: "my-image@sha256:abc123",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
				{Field: "repository", Values: []string{"*"}},
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
	config.Image = strings.TrimSpace(config.Image)
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
	if config.Image == "" {
		return fmt.Errorf("image is required")
	}
	return nil
}

func (c *GetArtifact) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	if config.Location == "" || config.Repository == "" || config.Image == "" {
		return ctx.ExecutionState.Fail("error", "location, repository, and image are required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	encodedImage := url.PathEscape(config.Image)
	apiURL := fmt.Sprintf(
		"%s/projects/%s/locations/%s/repositories/%s/dockerImages/%s",
		artifactRegistryBaseURL,
		client.ProjectID(),
		config.Location,
		config.Repository,
		encodedImage,
	)

	responseBody, err := client.GetURL(context.Background(), apiURL)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get artifact: %v", err))
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
