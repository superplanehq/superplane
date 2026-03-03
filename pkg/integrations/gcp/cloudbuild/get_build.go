package cloudbuild

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
	getBuildPayloadType   = "gcp.cloudbuild.build"
	getBuildOutputChannel = "default"
)

type GetBuild struct{}

type GetBuildConfiguration struct {
	BuildID   string `json:"buildId" mapstructure:"buildId"`
	ProjectID string `json:"projectId" mapstructure:"projectId"`
}

func (c *GetBuild) Name() string {
	return "gcp.cloudbuild.getBuild"
}

func (c *GetBuild) Label() string {
	return "Get Build"
}

func (c *GetBuild) Description() string {
	return "Retrieve a Cloud Build build by ID"
}

func (c *GetBuild) Documentation() string {
	return `Retrieves the details of a specific Google Cloud Build build.

## Configuration

- **Build ID** (required): The ID or full resource name of the Cloud Build build to retrieve.
- **Project ID Override**: Override the GCP project ID from the integration.

## Output

The full Build resource, including ` + "`id`" + `, ` + "`status`" + ` (SUCCESS, FAILURE, WORKING, QUEUED, etc.), ` + "`logUrl`" + `, ` + "`steps`" + `, ` + "`images`" + `, ` + "`createTime`" + `, ` + "`finishTime`" + `, and more.`
}

func (c *GetBuild) Icon() string  { return "gcp" }
func (c *GetBuild) Color() string { return "gray" }

func (c *GetBuild) ExampleOutput() map[string]any {
	return map[string]any{
		"id":         "12345678-abcd-1234-5678-abcdef012345",
		"projectId":  "my-project",
		"status":     "SUCCESS",
		"logUrl":     "https://console.cloud.google.com/cloud-build/builds/12345678-abcd-1234-5678-abcdef012345",
		"createTime": "2025-01-01T00:00:00Z",
		"finishTime": "2025-01-01T00:05:00Z",
	}
}

func (c *GetBuild) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project ID Override",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the GCP project ID from the integration. Leave empty to use the integration's project.",
		},
		{
			Name:        "buildId",
			Label:       "Build ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Cloud Build build to retrieve.",
			Placeholder: "Select a build",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeBuild,
					Parameters: []configuration.ParameterRef{
						{Name: "projectId", ValueFrom: &configuration.ParameterValueFrom{Field: "projectId"}},
					},
				},
			},
		},
	}
}

func (c *GetBuild) Setup(ctx core.SetupContext) error {
	var config GetBuildConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.BuildID) == "" {
		return fmt.Errorf("buildId is required")
	}
	return nil
}

func (c *GetBuild) Execute(ctx core.ExecutionContext) error {
	var config GetBuildConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	buildID := strings.TrimSpace(config.BuildID)
	if buildID == "" {
		return ctx.ExecutionState.Fail("error", "buildId is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}

	url := buildGetURL(projectID, buildID, buildID)
	responseBody, err := client.GetURL(context.Background(), url)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get build: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse response: %v", err))
	}

	return ctx.ExecutionState.Emit(getBuildOutputChannel, getBuildPayloadType, []any{result})
}

func (c *GetBuild) Actions() []core.Action                  { return nil }
func (c *GetBuild) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetBuild) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *GetBuild) Cancel(_ core.ExecutionContext) error { return nil }
func (c *GetBuild) Cleanup(_ core.SetupContext) error    { return nil }
func (c *GetBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
