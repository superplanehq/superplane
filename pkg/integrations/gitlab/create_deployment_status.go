package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_create_deployment_status.json
var exampleOutputCreateDeploymentStatus []byte

type CreateDeploymentStatus struct{}

type CreateDeploymentStatusConfiguration struct {
	Project      string `mapstructure:"project"`
	DeploymentID string `mapstructure:"deploymentId"`
	Status       string `mapstructure:"status"`
}

func (c *CreateDeploymentStatus) Name() string {
	return "gitlab.createDeploymentStatus"
}

func (c *CreateDeploymentStatus) Label() string {
	return "Create Deployment Status"
}

func (c *CreateDeploymentStatus) Description() string {
	return "Update the status of a GitLab deployment"
}

func (c *CreateDeploymentStatus) Documentation() string {
	return `The Create Deployment Status component updates the status of an existing GitLab deployment.

## Use Cases

- **Rollout progress**: Transition a deployment from running to success or failed as your workflow completes
- **Deployment tracking**: Keep an environment's deployment history in sync with an external rollout
- **Follow-up to Create Deployment**: Reference the deployment created upstream and mark its final outcome

## Configuration

- **Project** (required): The GitLab project containing the deployment
- **Deployment ID** (required): The ID of the deployment to update. Supports expressions, e.g. ` + "`{{ $['Create Deployment'].id }}`" + `
- **Status** (required): The new deployment status (running, success, failed, canceled)

## Output

Returns the updated deployment object, including:
- **id**: The deployment ID
- **status**: The new deployment status
- **environment**: The environment the deployment targets

## Requirements

The connected user needs at least the **Developer** role on the project, and for protected environments must be in the environment's **Allowed to deploy** list.`
}

func (c *CreateDeploymentStatus) Icon() string {
	return "gitlab"
}

func (c *CreateDeploymentStatus) Color() string {
	return "orange"
}

func (c *CreateDeploymentStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDeploymentStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateDeploymentStatus, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *CreateDeploymentStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "deploymentId",
			Label:       "Deployment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the deployment to update. Supports expressions, e.g. {{ $['Create Deployment'].id }}",
		},
		{
			Name:     "status",
			Label:    "Status",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  DeploymentStatusSuccess,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: deploymentStatusOptions(),
				},
			},
		},
	}
}

func (c *CreateDeploymentStatus) Setup(ctx core.SetupContext) error {
	var config CreateDeploymentStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if strings.TrimSpace(config.DeploymentID) == "" {
		return fmt.Errorf("deployment ID is required")
	}

	if strings.TrimSpace(config.Status) == "" {
		return fmt.Errorf("status is required")
	}

	if !slices.Contains(deploymentStatuses, config.Status) {
		return fmt.Errorf("invalid status %q: must be one of running, success, failed, canceled", config.Status)
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *CreateDeploymentStatus) Execute(ctx core.ExecutionContext) error {
	var config CreateDeploymentStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	deploymentID, err := parseDeploymentID(config.DeploymentID)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	deployment, err := client.UpdateDeployment(context.Background(), config.Project, deploymentID, &UpdateDeploymentRequest{
		Status: config.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeploymentPayloadType,
		[]any{deployment},
	)
}

func (c *CreateDeploymentStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDeploymentStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateDeploymentStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDeploymentStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateDeploymentStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateDeploymentStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// parseDeploymentID converts the configured deployment ID (which may come from an
// expression that resolves to a number) into an int.
func parseDeploymentID(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("deployment ID is required")
	}

	if id, err := strconv.Atoi(trimmed); err == nil {
		return id, nil
	}

	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return int(f), nil
	}

	return 0, fmt.Errorf("invalid deployment ID %q: must be a number", value)
}
