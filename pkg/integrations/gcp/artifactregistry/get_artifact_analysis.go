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
	getArtifactAnalysisPayloadType   = "gcp.artifactregistry.artifactAnalysis"
	getArtifactAnalysisOutputChannel = "default"
)

type GetArtifactAnalysis struct{}

type GetArtifactAnalysisConfiguration struct {
	ResourceURL string `json:"resourceUrl" mapstructure:"resourceUrl"`
}

func (c *GetArtifactAnalysis) Name() string {
	return "gcp.artifactregistry.getArtifactAnalysis"
}

func (c *GetArtifactAnalysis) Label() string {
	return "Artifact Registry • Get Artifact Analysis"
}

func (c *GetArtifactAnalysis) Description() string {
	return "Retrieve vulnerability analysis for an Artifact Registry image"
}

func (c *GetArtifactAnalysis) Documentation() string {
	return `Retrieves vulnerability occurrences for a container image from GCP Container Analysis (Artifact Analysis).

## Configuration

- **Resource URL** (required): The full resource URL of the image, in the format ` + "`https://LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/IMAGE@sha256:DIGEST`" + `.

## Output

The list of vulnerability occurrences for the specified image, including severity, package name, description, and fix information.

## Notes

- The Container Analysis API must be enabled in your project.
- Automatic scanning must be enabled for the Artifact Registry repository, or you must have previously triggered a scan via the Analyze Artifact component.
- The service account must have ` + "`roles/containeranalysis.occurrences.viewer`" + `.`
}

func (c *GetArtifactAnalysis) Icon() string  { return "gcp" }
func (c *GetArtifactAnalysis) Color() string { return "gray" }

func (c *GetArtifactAnalysis) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetArtifactAnalysis) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceUrl",
			Label:       "Resource URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full resource URL of the container image.",
			Placeholder: "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
		},
	}
}

func decodeGetArtifactAnalysisConfiguration(raw any) (GetArtifactAnalysisConfiguration, error) {
	var config GetArtifactAnalysisConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return GetArtifactAnalysisConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ResourceURL = strings.TrimSpace(config.ResourceURL)
	return config, nil
}

func (c *GetArtifactAnalysis) Setup(ctx core.SetupContext) error {
	config, err := decodeGetArtifactAnalysisConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.ResourceURL == "" {
		return fmt.Errorf("resourceUrl is required")
	}
	return nil
}

func (c *GetArtifactAnalysis) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetArtifactAnalysisConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	if config.ResourceURL == "" {
		return ctx.ExecutionState.Fail("error", "resourceUrl is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	filter := fmt.Sprintf(`kind="VULNERABILITY" AND resourceUrl="%s"`, config.ResourceURL)
	apiURL := fmt.Sprintf(
		"%s/projects/%s/occurrences?filter=%s",
		containerAnalysisBaseURL,
		client.ProjectID(),
		url.QueryEscape(filter),
	)

	responseBody, err := client.GetURL(context.Background(), apiURL)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get artifact analysis: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse response: %v", err))
	}

	return ctx.ExecutionState.Emit(getArtifactAnalysisOutputChannel, getArtifactAnalysisPayloadType, []any{result})
}

func (c *GetArtifactAnalysis) Actions() []core.Action                  { return nil }
func (c *GetArtifactAnalysis) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetArtifactAnalysis) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *GetArtifactAnalysis) Cancel(_ core.ExecutionContext) error { return nil }
func (c *GetArtifactAnalysis) Cleanup(_ core.SetupContext) error    { return nil }
func (c *GetArtifactAnalysis) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
