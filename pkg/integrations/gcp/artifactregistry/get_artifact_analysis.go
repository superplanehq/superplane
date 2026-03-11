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
	InputMode   string `json:"inputMode" mapstructure:"inputMode"`
	ResourceURL string `json:"resourceUrl" mapstructure:"resourceUrl"`
	Location    string `json:"location" mapstructure:"location"`
	Repository  string `json:"repository" mapstructure:"repository"`
	Package     string `json:"package" mapstructure:"package"`
	Version     string `json:"version" mapstructure:"version"`
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

Provide either a **Resource URL** or the four fields below:

- **Resource URL**: Full resource URL of the image (e.g. ` + "`https://us-central1-docker.pkg.dev/project/repo/image@sha256:abc`" + `). Use this to pass a digest directly from an upstream event such as On Artifact Push.
- **Location**: The GCP region where the repository is located.
- **Repository**: The Artifact Registry repository containing the artifact.
- **Package**: The package (image) within the repository.
- **Version**: The version (digest) to query.

## Output

An analysis summary for the artifact, including:
- ` + "`resourceUri`" + `: The analyzed artifact URI
- ` + "`scanStatus`" + `: Discovery scan status (if available)
- Severity counts: ` + "`critical`" + `, ` + "`high`" + `, ` + "`medium`" + `, ` + "`low`" + `
- ` + "`vulnerabilities`" + `: Total vulnerability occurrences
- ` + "`fixAvailable`" + `: Count of vulnerabilities with fixes

## Notes

- The **Container Analysis API** (` + "`containeranalysis.googleapis.com`" + `) must be enabled.
- The service account needs ` + "`roles/containeranalysis.occurrences.viewer`" + `.
- This summarizes existing occurrences for the selected artifact.`
}

func (c *GetArtifactAnalysis) Icon() string  { return "gcp" }
func (c *GetArtifactAnalysis) Color() string { return "gray" }

func (c *GetArtifactAnalysis) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetArtifactAnalysis) Configuration() []configuration.Field {
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
			Description: "Select the package (image) to query.",
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
			Description: "Select the version (digest) to query.",
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

func decodeGetArtifactAnalysisConfiguration(raw any) (GetArtifactAnalysisConfiguration, error) {
	var config GetArtifactAnalysisConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return GetArtifactAnalysisConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ResourceURL = sanitizeConfigValue(config.ResourceURL)
	config.Location = strings.TrimSpace(config.Location)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Package = strings.TrimSpace(config.Package)
	config.Version = strings.TrimSpace(config.Version)
	return config, nil
}

// sanitizeConfigValue trims whitespace and clears nil-expression artifacts.
func sanitizeConfigValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "<nil>" || s == "nil" {
		return ""
	}
	return s
}

func (c *GetArtifactAnalysis) Setup(ctx core.SetupContext) error {
	config, err := decodeGetArtifactAnalysisConfiguration(ctx.Configuration)
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

func (c *GetArtifactAnalysis) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetArtifactAnalysisConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	useURLMode := config.InputMode == "url" || config.InputMode == ""
	if useURLMode {
		if config.ResourceURL == "" {
			return ctx.ExecutionState.Fail("error", "resourceUrl is required in url mode")
		}
	} else {
		if config.Location == "" {
			return ctx.ExecutionState.Fail("error", "location is required")
		}
		if config.Repository == "" {
			return ctx.ExecutionState.Fail("error", "repository is required")
		}
		if config.Package == "" {
			return ctx.ExecutionState.Fail("error", "package is required")
		}
		if config.Version == "" {
			return ctx.ExecutionState.Fail("error", "version is required")
		}
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	var resourceURI string
	if useURLMode {
		resourceURI = config.ResourceURL
	} else {
		resourceURI = fmt.Sprintf("https://%s-docker.pkg.dev/%s/%s/%s@%s", config.Location, projectID, config.Repository, config.Package, config.Version)
	}
	filter := buildOccurrenceFilter(resourceURI)
	reqURL := listOccurrencesURL(projectID, filter)

	summary := &analysisSummary{ResourceURI: resourceURI}

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
			summary.accumulate(occ)
		}

		if resp.NextPageToken == "" {
			break
		}

		reqURL = addPageTokenToURL(reqURL, resp.NextPageToken)
	}

	return ctx.ExecutionState.Emit(getArtifactAnalysisOutputChannel, getArtifactAnalysisPayloadType, []any{summary})
}

func buildOccurrenceFilter(resourceURI string) string {
	var parts []string

	if resourceURI != "" {
		parts = append(parts, fmt.Sprintf("resourceUrl=%q", resourceURI))
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

type analysisSummary struct {
	ResourceURI     string `json:"resourceUri"`
	ScanStatus      string `json:"scanStatus,omitempty"`
	Vulnerabilities int    `json:"vulnerabilities"`
	Critical        int    `json:"critical"`
	High            int    `json:"high"`
	Medium          int    `json:"medium"`
	Low             int    `json:"low"`
	FixAvailable    int    `json:"fixAvailable"`
}

func (s *analysisSummary) accumulate(occ map[string]any) {
	kind, _ := occ["kind"].(string)
	switch kind {
	case "DISCOVERY":
		if disc, ok := occ["discovery"].(map[string]any); ok {
			if status, ok := disc["analysisStatus"].(string); ok {
				s.ScanStatus = status
			}
		}
	case "VULNERABILITY":
		s.Vulnerabilities++
		if vuln, ok := occ["vulnerability"].(map[string]any); ok {
			switch vuln["effectiveSeverity"] {
			case "CRITICAL":
				s.Critical++
			case "HIGH":
				s.High++
			case "MEDIUM":
				s.Medium++
			case "LOW":
				s.Low++
			}
			if vuln["fixAvailable"] == true {
				s.FixAvailable++
			}
		}
	}
}

func (c *GetArtifactAnalysis) Actions() []core.Action                  { return nil }
func (c *GetArtifactAnalysis) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetArtifactAnalysis) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *GetArtifactAnalysis) Cancel(_ core.ExecutionContext) error { return nil }
func (c *GetArtifactAnalysis) Cleanup(_ core.SetupContext) error    { return nil }
func (c *GetArtifactAnalysis) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
