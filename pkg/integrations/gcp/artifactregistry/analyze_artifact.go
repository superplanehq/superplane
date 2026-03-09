package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	analyzeArtifactPayloadType          = "gcp.artifactregistry.vulnerabilities"
	analyzeArtifactPassedOutputChannel  = "passed"
	analyzeArtifactFailedOutputChannel  = "failed"
	analyzeArtifactPollAction           = "poll"
	analyzeArtifactPollInterval         = 30 * time.Second
	analyzeArtifactExecutionKV          = "scan_operation"
)

type AnalyzeArtifact struct{}

type AnalyzeArtifactConfiguration struct {
	Location   string `json:"location" mapstructure:"location"`
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
	Version    string `json:"version" mapstructure:"version"`
}

type AnalyzeArtifactExecutionMetadata struct {
	OperationName string `json:"operationName,omitempty" mapstructure:"operationName,omitempty"`
	ScanName      string `json:"scanName,omitempty" mapstructure:"scanName,omitempty"`
}

func (c *AnalyzeArtifact) Name() string {
	return "gcp.artifactregistry.analyzeArtifact"
}

func (c *AnalyzeArtifact) Label() string {
	return "Artifact Registry • Analyze Artifact"
}

func (c *AnalyzeArtifact) Description() string {
	return "Trigger on-demand vulnerability analysis for a container image and wait for results"
}

func (c *AnalyzeArtifact) Documentation() string {
	return `Triggers Google On Demand Scanning to perform vulnerability analysis on a container image and waits for the scan to complete.

## Configuration

- **Resource URI** (required): The full resource URI of the container image to scan (e.g. ` + "`us-central1-docker.pkg.dev/project/repo/image@sha256:...`" + `). You can use template variables to pass the digest from an upstream On Artifact Push event.
- **Location** (required): The GCP region where the scan should run (must match the image location).

## Output

The vulnerability findings from the scan, including a list of ` + "`vulnerabilities`" + ` each with severity, CVE ID, affected package, and fix availability.

## Output Channels

- **Passed**: Emitted when the scan completes successfully (even if vulnerabilities are found).
- **Failed**: Emitted when the scan fails or encounters an error.

## Notes

- Only container images stored in Artifact Registry or Container Registry are supported.
- The ` + "`On Demand Scanning API`" + ` (` + "`ondemandscanning.googleapis.com`" + `) must be enabled in your project.
- The service account needs ` + "`roles/ondemandscanning.admin`" + ` or ` + "`roles/ondemandscanning.editor`" + `.`
}

func (c *AnalyzeArtifact) Icon() string  { return "gcp" }
func (c *AnalyzeArtifact) Color() string { return "gray" }

func (c *AnalyzeArtifact) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: analyzeArtifactPassedOutputChannel, Label: "Passed"},
		{Name: analyzeArtifactFailedOutputChannel, Label: "Failed"},
	}
}

func (c *AnalyzeArtifact) Configuration() []configuration.Field {
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
			Description: "Select the package (image) to scan.",
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
			Description: "Select the version (digest) to scan.",
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

func decodeAnalyzeArtifactConfiguration(raw any) (AnalyzeArtifactConfiguration, error) {
	var config AnalyzeArtifactConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return AnalyzeArtifactConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
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

func (c *AnalyzeArtifact) Execute(ctx core.ExecutionContext) error {
	config, err := decodeAnalyzeArtifactConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	resourceURI := fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s@%s", config.Location, projectID, config.Repository, config.Package, config.Version)
	url := analyzePackagesURL(projectID, config.Location)
	body := map[string]any{
		"resourceUri": resourceURI,
	}

	responseBody, err := client.PostURL(context.Background(), url, body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to start vulnerability scan: %v", err))
	}

	var operation struct {
		Name     string         `json:"name"`
		Done     bool           `json:"done"`
		Error    map[string]any `json:"error"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(responseBody, &operation); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse operation response: %v", err))
	}

	if operation.Done {
		if len(operation.Error) > 0 {
			return ctx.ExecutionState.Emit(analyzeArtifactFailedOutputChannel, analyzeArtifactPayloadType, []any{operation.Error})
		}
		return c.fetchAndEmitVulnerabilities(ctx.ExecutionState, client, operation.Metadata)
	}

	metadata := AnalyzeArtifactExecutionMetadata{OperationName: operation.Name}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to store operation metadata: %v", err))
	}

	if err := ctx.ExecutionState.SetKV(analyzeArtifactExecutionKV, operation.Name); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to track scan operation: %v", err))
	}

	return ctx.Requests.ScheduleActionCall(analyzeArtifactPollAction, map[string]any{}, analyzeArtifactPollInterval)
}

func (c *AnalyzeArtifact) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata AnalyzeArtifactExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.OperationName == "" {
		return fmt.Errorf("operation name is missing from metadata")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	url := getOperationURL(metadata.OperationName)
	responseBody, err := client.GetURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to poll operation: %w", err)
	}

	var operation struct {
		Name     string         `json:"name"`
		Done     bool           `json:"done"`
		Error    map[string]any `json:"error"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(responseBody, &operation); err != nil {
		return fmt.Errorf("failed to parse operation response: %w", err)
	}

	if !operation.Done {
		return ctx.Requests.ScheduleActionCall(analyzeArtifactPollAction, map[string]any{}, analyzeArtifactPollInterval)
	}

	if len(operation.Error) > 0 {
		return ctx.ExecutionState.Emit(analyzeArtifactFailedOutputChannel, analyzeArtifactPayloadType, []any{operation.Error})
	}

	return c.fetchAndEmitVulnerabilities(ctx.ExecutionState, client, operation.Metadata)
}

func (c *AnalyzeArtifact) fetchAndEmitVulnerabilities(
	executionState core.ExecutionStateContext,
	client Client,
	operationMetadata map[string]any,
) error {
	scanName, _ := operationMetadata["scan"].(string)
	if scanName == "" {
		return executionState.Emit(analyzeArtifactPassedOutputChannel, analyzeArtifactPayloadType, []any{map[string]any{
			"vulnerabilities": []any{},
		}})
	}

	url := listVulnerabilitiesURL(scanName)
	responseBody, err := client.GetURL(context.Background(), url)
	if err != nil {
		return executionState.Emit(analyzeArtifactFailedOutputChannel, analyzeArtifactPayloadType, []any{map[string]any{
			"error": err.Error(),
		}})
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return executionState.Emit(analyzeArtifactFailedOutputChannel, analyzeArtifactPayloadType, []any{map[string]any{
			"error": fmt.Sprintf("failed to parse vulnerabilities: %v", err),
		}})
	}

	result["scan"] = scanName
	return executionState.Emit(analyzeArtifactPassedOutputChannel, analyzeArtifactPayloadType, []any{result})
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

func (c *AnalyzeArtifact) OnIntegrationMessage(_ core.IntegrationMessageContext) error { return nil }

func (c *AnalyzeArtifact) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *AnalyzeArtifact) Cancel(_ core.ExecutionContext) error { return nil }
func (c *AnalyzeArtifact) Cleanup(_ core.SetupContext) error    { return nil }
func (c *AnalyzeArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
