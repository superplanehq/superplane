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
	analyzeArtifactPayloadType   = "gcp.artifactregistry.artifactAnalysis"
	analyzeArtifactOutputChannel = "default"
)

type AnalyzeArtifact struct{}

type AnalyzeArtifactConfiguration struct {
	ResourceURL string `json:"resourceUrl" mapstructure:"resourceUrl"`
	Location    string `json:"location" mapstructure:"location"`
	Repository  string `json:"repository" mapstructure:"repository"`
	Package     string `json:"package" mapstructure:"package"`
	Version     string `json:"version" mapstructure:"version"`
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
	return `Queries Container Analysis for vulnerability occurrences on a container image and emits the results immediately.

## Configuration

Provide either a **Resource URL** or the four fields below:

- **Resource URL**: Full resource URL of the image (e.g. from an On Artifact Push event). When provided, the fields below are not required.
- **Location**: The GCP region where the repository is located.
- **Repository**: The Artifact Registry repository containing the artifact.
- **Package**: The package (image) within the repository.
- **Version**: The version (digest or tag) to analyze.

## Output

The list of vulnerability occurrences for the image. May be empty if the image has not been scanned yet.

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
			Required:    false,
			Description: "Full resource URL of the container image to analyze (e.g. from an On Artifact Push event). When provided, the fields below are not required.",
			Placeholder: "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the Artifact Registry region. Required if Resource URL is not provided.",
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
			Description: "Select the Artifact Registry repository. Required if Resource URL is not provided.",
			VisibilityConditions: []configuration.VisibilityCondition{
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
			Description: "Select the package (image) to analyze. Required if Resource URL is not provided.",
			VisibilityConditions: []configuration.VisibilityCondition{
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
			Description: "Select the version (digest or tag) to analyze. Required if Resource URL is not provided.",
			VisibilityConditions: []configuration.VisibilityCondition{
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

func decodeAnalyzeArtifactConfiguration(raw any) (AnalyzeArtifactConfiguration, error) {
	var config AnalyzeArtifactConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return AnalyzeArtifactConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ResourceURL = strings.TrimSpace(config.ResourceURL)
	config.Location = strings.TrimSpace(config.Location)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Package = strings.TrimSpace(config.Package)
	config.Version = strings.TrimSpace(config.Version)
	return config, nil
}

func (c *AnalyzeArtifact) Setup(ctx core.SetupContext) error {
	config, err := decodeAnalyzeArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.ResourceURL != "" {
		_, _, _, _, err := parseArtifactResourceURL(config.ResourceURL)
		return err
	}
	if config.Location == "" {
		return fmt.Errorf("location is required (or provide resourceUrl)")
	}
	if config.Repository == "" {
		return fmt.Errorf("repository is required (or provide resourceUrl)")
	}
	if config.Package == "" {
		return fmt.Errorf("package is required (or provide resourceUrl)")
	}
	if config.Version == "" {
		return fmt.Errorf("version is required (or provide resourceUrl)")
	}
	return nil
}

func (c *AnalyzeArtifact) Execute(ctx core.ExecutionContext) error {
	config, err := decodeAnalyzeArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	resourceURL := config.ResourceURL
	if resourceURL == "" {
		projectID := client.ProjectID()
		resourceURL = fmt.Sprintf("https://%s-docker.pkg.dev/%s/%s/%s@%s",
			config.Location, projectID, config.Repository, config.Package, config.Version)
	}

	result, _, err := fetchVulnerabilityOccurrences(client, resourceURL)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to query vulnerability occurrences: %v", err))
	}

	return ctx.ExecutionState.Emit(analyzeArtifactOutputChannel, analyzeArtifactPayloadType, []any{result})
}

func (c *AnalyzeArtifact) Actions() []core.Action                  { return nil }
func (c *AnalyzeArtifact) HandleAction(_ core.ActionContext) error { return nil }

func fetchVulnerabilityOccurrences(client Client, resourceURL string) (map[string]any, bool, error) {
	filter := fmt.Sprintf(`kind="VULNERABILITY" AND resourceUrl="%s"`, resourceURL)
	reqURL := fmt.Sprintf(
		"%s/projects/%s/occurrences?filter=%s",
		containerAnalysisBaseURL,
		client.ProjectID(),
		url.QueryEscape(filter),
	)

	allOccurrences := make([]any, 0)

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
