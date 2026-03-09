package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	analyzeArtifactPayloadType   = "gcp.artifactregistry.artifactAnalysis"
	analyzeArtifactOutputChannel = "default"
	analyzeArtifactPollAction    = "poll"
	analyzeArtifactPollInterval  = 30 * time.Second
)

type AnalyzeArtifact struct{}

type AnalyzeArtifactConfiguration struct {
	ResourceURL string `json:"resourceUrl" mapstructure:"resourceUrl"`
}

type AnalyzeArtifactMetadata struct {
	ResourceURL string `json:"resourceUrl" mapstructure:"resourceUrl"`
}

func (c *AnalyzeArtifact) Name() string {
	return "gcp.artifactregistry.analyzeArtifact"
}

func (c *AnalyzeArtifact) Label() string {
	return "Artifact Registry • Analyze Artifact"
}

func (c *AnalyzeArtifact) Description() string {
	return "Analyze an Artifact Registry image for vulnerabilities and wait for results"
}

func (c *AnalyzeArtifact) Documentation() string {
	return `Queries Container Analysis for vulnerability occurrences on a container image and waits until scan results are available.

## Configuration

- **Resource URL** (required): The full resource URL of the image, in the format ` + "`https://LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/IMAGE@sha256:DIGEST`" + `.

## Behavior

1. Queries the Container Analysis API for existing vulnerability occurrences for the specified image.
2. If occurrences are found, emits them immediately.
3. If no occurrences are found (scan may still be in progress), polls every 30 seconds until results appear.

## Output

The list of vulnerability occurrences, including severity, package name, and remediation information.

## Notes

- Automatic vulnerability scanning must be enabled in your Artifact Registry repository settings.
- The Container Analysis API must be enabled in your project.
- The service account must have ` + "`roles/containeranalysis.occurrences.viewer`" + `.`
}

func (c *AnalyzeArtifact) Icon() string  { return "gcp" }
func (c *AnalyzeArtifact) Color() string { return "gray" }

func (c *AnalyzeArtifact) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AnalyzeArtifact) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "resourceUrl",
			Label:       "Resource URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full resource URL of the container image to analyze.",
			Placeholder: "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
		},
	}
}

func decodeAnalyzeArtifactConfiguration(raw any) (AnalyzeArtifactConfiguration, error) {
	var config AnalyzeArtifactConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return AnalyzeArtifactConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ResourceURL = strings.TrimSpace(config.ResourceURL)
	return config, nil
}

func (c *AnalyzeArtifact) Setup(ctx core.SetupContext) error {
	config, err := decodeAnalyzeArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.ResourceURL == "" {
		return fmt.Errorf("resourceUrl is required")
	}
	return nil
}

func (c *AnalyzeArtifact) Execute(ctx core.ExecutionContext) error {
	config, err := decodeAnalyzeArtifactConfiguration(ctx.Configuration)
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

	result, hasOccurrences, err := fetchVulnerabilityOccurrences(client, config.ResourceURL)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to query vulnerability occurrences: %v", err))
	}

	if hasOccurrences {
		return ctx.ExecutionState.Emit(analyzeArtifactOutputChannel, analyzeArtifactPayloadType, []any{result})
	}

	if err := ctx.Metadata.Set(AnalyzeArtifactMetadata{ResourceURL: config.ResourceURL}); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to store metadata: %v", err))
	}

	return ctx.Requests.ScheduleActionCall(analyzeArtifactPollAction, map[string]any{}, analyzeArtifactPollInterval)
}

func (c *AnalyzeArtifact) Actions() []core.Action {
	return []core.Action{
		{Name: analyzeArtifactPollAction, UserAccessible: false},
	}
}

func (c *AnalyzeArtifact) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case analyzeArtifactPollAction:
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *AnalyzeArtifact) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata AnalyzeArtifactMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	result, hasOccurrences, err := fetchVulnerabilityOccurrences(client, metadata.ResourceURL)
	if err != nil {
		return fmt.Errorf("failed to query vulnerability occurrences: %w", err)
	}

	if !hasOccurrences {
		return ctx.Requests.ScheduleActionCall(analyzeArtifactPollAction, map[string]any{}, analyzeArtifactPollInterval)
	}

	return ctx.ExecutionState.Emit(analyzeArtifactOutputChannel, analyzeArtifactPayloadType, []any{result})
}

func fetchVulnerabilityOccurrences(client Client, resourceURL string) (map[string]any, bool, error) {
	filter := fmt.Sprintf(`kind="VULNERABILITY" AND resourceUrl="%s"`, resourceURL)
	reqURL := fmt.Sprintf(
		"%s/projects/%s/occurrences?filter=%s",
		containerAnalysisBaseURL,
		client.ProjectID(),
		url.QueryEscape(filter),
	)

	var allOccurrences []any

	for {
		responseBody, err := client.GetURL(context.Background(), reqURL)
		if err != nil {
			return nil, false, err
		}

		var resp struct {
			Occurrences   []map[string]any `json:"occurrences"`
			NextPageToken string           `json:"nextPageToken"`
		}
		if err := json.Unmarshal(responseBody, &resp); err != nil {
			return nil, false, fmt.Errorf("failed to parse occurrences response: %w", err)
		}

		for _, occ := range resp.Occurrences {
			allOccurrences = append(allOccurrences, occ)
		}

		if resp.NextPageToken == "" {
			break
		}

		reqURL = addPageTokenToURL(reqURL, resp.NextPageToken)
	}

	result := map[string]any{
		"occurrences": allOccurrences,
		"resourceUri": resourceURL,
	}

	return result, len(allOccurrences) > 0, nil
}

func (c *AnalyzeArtifact) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *AnalyzeArtifact) Cancel(_ core.ExecutionContext) error { return nil }
func (c *AnalyzeArtifact) Cleanup(_ core.SetupContext) error    { return nil }
func (c *AnalyzeArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
