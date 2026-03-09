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
	getArtifactAnalysisPayloadType   = "gcp.containeranalysis.occurrences"
	getArtifactAnalysisOutputChannel = "default"
)

type GetArtifactAnalysis struct{}

type GetArtifactAnalysisConfiguration struct {
	Location   string   `json:"location" mapstructure:"location"`
	Repository string   `json:"repository" mapstructure:"repository"`
	Package    string   `json:"package" mapstructure:"package"`
	Version    string   `json:"version" mapstructure:"version"`
	Kinds      []string `json:"kinds" mapstructure:"kinds"`
}

func (c *GetArtifactAnalysis) Name() string {
	return "gcp.artifactregistry.getArtifactAnalysis"
}

func (c *GetArtifactAnalysis) Label() string {
	return "Artifact Registry • Get Artifact Analysis"
}

func (c *GetArtifactAnalysis) Description() string {
	return "Retrieve Container Analysis occurrences (vulnerabilities, build provenance, attestations) for an artifact"
}

func (c *GetArtifactAnalysis) Documentation() string {
	return `Retrieves existing Container Analysis occurrences for an artifact from Google Container Analysis.

## Configuration

- **Resource URI** (required): The resource URI of the artifact to query (e.g. ` + "`https://us-central1-docker.pkg.dev/project/repo/image@sha256:...`" + `).
- **Occurrence Kinds**: Optional filter by occurrence kind (VULNERABILITY, BUILD, ATTESTATION, SBOM). Leave empty to retrieve all kinds.

## Output

A list of Container Analysis occurrences for the artifact. Each occurrence includes ` + "`kind`" + `, ` + "`resourceUri`" + `, ` + "`noteName`" + `, and the occurrence-specific data.

## Notes

- The **Container Analysis API** (` + "`containeranalysis.googleapis.com`" + `) must be enabled.
- The service account needs ` + "`roles/containeranalysis.occurrences.viewer`" + `.
- This retrieves existing occurrences. To trigger a new analysis, use the **Analyze Artifact** component.`
}

func (c *GetArtifactAnalysis) Icon() string  { return "gcp" }
func (c *GetArtifactAnalysis) Color() string { return "gray" }

func (c *GetArtifactAnalysis) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetArtifactAnalysis) Configuration() []configuration.Field {
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
			Description: "Select the package (image) to query.",
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
			Description: "Select the version (digest) to query.",
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
		{
			Name:        "kinds",
			Label:       "Occurrence Kinds",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by occurrence kind. Leave empty to retrieve all kinds.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Vulnerability", Value: occurrenceKindVulnerability},
						{Label: "Build Provenance", Value: occurrenceKindBuild},
						{Label: "Attestation", Value: occurrenceKindAttestation},
						{Label: "SBOM", Value: occurrenceKindSBOM},
					},
				},
			},
		},
	}
}

func decodeGetArtifactAnalysisConfiguration(raw any) (GetArtifactAnalysisConfiguration, error) {
	var config GetArtifactAnalysisConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return GetArtifactAnalysisConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Location = strings.TrimSpace(config.Location)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Package = strings.TrimSpace(config.Package)
	config.Version = strings.TrimSpace(config.Version)
	return config, nil
}

func (c *GetArtifactAnalysis) Setup(ctx core.SetupContext) error {
	config, err := decodeGetArtifactAnalysisConfiguration(ctx.Configuration)
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

func (c *GetArtifactAnalysis) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetArtifactAnalysisConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	resourceURI := fmt.Sprintf("https://%s-docker.pkg.dev/%s/%s/%s@%s", config.Location, projectID, config.Repository, config.Package, config.Version)
	filter := buildOccurrenceFilter(config, resourceURI)
	reqURL := listOccurrencesURL(projectID, filter)

	var allOccurrences []any

	for {
		responseBody, err := client.GetURL(context.Background(), reqURL)
		if err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to list occurrences: %v", err))
		}

		var resp struct {
			Occurrences   []map[string]any `json:"occurrences"`
			NextPageToken string           `json:"nextPageToken"`
		}
		if err := json.Unmarshal(responseBody, &resp); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse occurrences: %v", err))
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
		"resourceUri": resourceURI,
	}

	return ctx.ExecutionState.Emit(getArtifactAnalysisOutputChannel, getArtifactAnalysisPayloadType, []any{result})
}

func buildOccurrenceFilter(config GetArtifactAnalysisConfiguration, resourceURI string) string {
	var parts []string

	if resourceURI != "" {
		parts = append(parts, fmt.Sprintf("resourceUrl=%q", resourceURI))
	}

	if len(config.Kinds) > 0 {
		kindFilter := make([]string, 0, len(config.Kinds))
		for _, k := range config.Kinds {
			kindFilter = append(kindFilter, fmt.Sprintf("kind=%q", k))
		}
		if len(kindFilter) == 1 {
			parts = append(parts, kindFilter[0])
		} else {
			parts = append(parts, "("+strings.Join(kindFilter, " OR ")+")")
		}
	}

	return strings.Join(parts, " AND ")
}

func addPageTokenToURL(baseURL, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return baseURL
	}
	encoded := url.Values{"pageToken": {token}}.Encode()
	if strings.Contains(baseURL, "?") {
		return baseURL + "&" + encoded
	}
	return baseURL + "?" + encoded
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
