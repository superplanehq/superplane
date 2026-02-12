package railway

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// Output channel names
	DeployedOutputChannel = "deployed"
	FailedOutputChannel   = "failed"
	CrashedOutputChannel  = "crashed"

	// Event type
	DeploymentPayloadType = "railway.deployment.finished"

	// Poll interval for checking deployment status
	DeploymentPollInterval = 15 * time.Second
)

// TriggerDeploy triggers a deployment and tracks its status until completion
type TriggerDeploy struct{}

type TriggerDeployConfiguration struct {
	Project     string `json:"project"     mapstructure:"project"`
	Service     string `json:"service"     mapstructure:"service"`
	Environment string `json:"environment" mapstructure:"environment"`
}

type TriggerDeployMetadata struct {
	Project     *ProjectInfo     `json:"project"     mapstructure:"project"`
	Service     *ServiceInfo     `json:"service"     mapstructure:"service"`
	Environment *EnvironmentInfo `json:"environment" mapstructure:"environment"`
}

// TriggerDeployExecutionMetadata tracks the deployment state during execution
type TriggerDeployExecutionMetadata struct {
	DeploymentID string `json:"deploymentId" mapstructure:"deploymentId"`
	Status       string `json:"status"       mapstructure:"status"`
	URL          string `json:"url"          mapstructure:"url"`
}

type ServiceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type EnvironmentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *TriggerDeploy) Name() string {
	return "railway.triggerDeploy"
}

func (c *TriggerDeploy) Label() string {
	return "Trigger Deploy"
}

func (c *TriggerDeploy) Description() string {
	return "Trigger a new deployment for a Railway service"
}

func (c *TriggerDeploy) Documentation() string {
	return `The Trigger Deploy component starts a new deployment for a Railway service and waits for it to complete.

## Use Cases

- **Deploy on merge**: Automatically deploy when code is merged to main
- **Scheduled deployments**: Deploy on a schedule (e.g., nightly releases)
- **Manual approval**: Deploy after approval in the workflow
- **Cross-service orchestration**: Deploy services in sequence

## How It Works

1. Triggers a new deployment via Railway's API
2. Polls for deployment status updates (Queued → Building → Deploying → Success/Failed)
3. Routes execution based on final deployment status:
   - **Deployed channel**: Deployment succeeded
   - **Failed channel**: Deployment failed
   - **Crashed channel**: Deployment crashed

## Configuration

- **Project**: Select the Railway project
- **Service**: Select the service to deploy
- **Environment**: Select the target environment (e.g., production, staging)

## Output Channels

- **Deployed**: Emitted when deployment succeeds
- **Failed**: Emitted when deployment fails
- **Crashed**: Emitted when deployment crashes`
}

func (c *TriggerDeploy) Icon() string {
	return "railway"
}

func (c *TriggerDeploy) Color() string {
	return "purple"
}

func (c *TriggerDeploy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Railway project containing the service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Service to deploy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "projectId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:        "environment",
			Label:       "Environment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Target environment for the deployment",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "environment",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "projectId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
}

func (c *TriggerDeploy) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  DeployedOutputChannel,
			Label: "Deployed",
		},
		{
			Name:  FailedOutputChannel,
			Label: "Failed",
		},
		{
			Name:  CrashedOutputChannel,
			Label: "Crashed",
		},
	}
}

func (c *TriggerDeploy) Setup(ctx core.SetupContext) error {
	config := TriggerDeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}
	if config.Service == "" {
		return fmt.Errorf("service is required")
	}
	if config.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	// Validate the resources exist
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Find service and environment in project
	var serviceName, environmentName string
	for _, svc := range project.Services {
		if svc.ID == config.Service {
			serviceName = svc.Name
			break
		}
	}
	for _, env := range project.Environments {
		if env.ID == config.Environment {
			environmentName = env.Name
			break
		}
	}

	if serviceName == "" {
		return fmt.Errorf("service not found in project")
	}
	if environmentName == "" {
		return fmt.Errorf("environment not found in project")
	}

	// Store metadata for display
	return ctx.Metadata.Set(TriggerDeployMetadata{
		Project:     &ProjectInfo{ID: project.ID, Name: project.Name},
		Service:     &ServiceInfo{ID: config.Service, Name: serviceName},
		Environment: &EnvironmentInfo{ID: config.Environment, Name: environmentName},
	})
}

func (c *TriggerDeploy) Execute(ctx core.ExecutionContext) error {
	config := TriggerDeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Trigger the deployment and get the deployment ID
	deploymentID, err := client.TriggerDeploy(config.Service, config.Environment)
	if err != nil {
		return fmt.Errorf("failed to trigger deployment: %w", err)
	}

	ctx.Logger.Infof(
		"Triggered deployment %s for service %s in environment %s",
		deploymentID,
		config.Service,
		config.Environment,
	)

	// Store deployment ID in execution metadata
	err = ctx.Metadata.Set(TriggerDeployExecutionMetadata{
		DeploymentID: deploymentID,
		Status:       DeploymentStatusQueued,
	})
	if err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	// Store deployment ID in KV for later retrieval
	err = ctx.ExecutionState.SetKV("deployment_id", deploymentID)
	if err != nil {
		return fmt.Errorf("failed to store deployment ID: %w", err)
	}

	// Schedule the first poll to check deployment status
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeploymentPollInterval)
}

func (c *TriggerDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TriggerDeploy) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *TriggerDeploy) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *TriggerDeploy) poll(ctx core.ActionContext) error {
	// Check if execution is already finished
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	// Get current execution metadata
	execMetadata := TriggerDeployExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &execMetadata); err != nil {
		return fmt.Errorf("failed to decode execution metadata: %w", err)
	}

	// If already in final state, nothing to do
	if IsDeploymentFinalStatus(execMetadata.Status) {
		return nil
	}

	// Create client to check deployment status
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get the latest deployment status
	deployment, err := client.GetDeployment(execMetadata.DeploymentID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("Failed to get deployment status, will retry")
		// Schedule another poll - don't fail on transient errors
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeploymentPollInterval)
	}

	ctx.Logger.Infof("Deployment %s status: %s", deployment.ID, deployment.Status)

	// Update metadata with current status
	execMetadata.Status = deployment.Status
	execMetadata.URL = deployment.URL
	if err := ctx.Metadata.Set(execMetadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// If not in final state, schedule another poll
	if !IsDeploymentFinalStatus(deployment.Status) {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeploymentPollInterval)
	}

	// Deployment reached final state - emit to appropriate channel
	config := TriggerDeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventData := map[string]any{
		"deploymentId": deployment.ID,
		"status":       deployment.Status,
		"url":          deployment.URL,
		"project":      config.Project,
		"service":      config.Service,
		"environment":  config.Environment,
	}

	switch deployment.Status {
	case DeploymentStatusSuccess:
		ctx.Logger.Info("Deployment succeeded")
		return ctx.ExecutionState.Emit(DeployedOutputChannel, DeploymentPayloadType, []any{eventData})
	case DeploymentStatusCrashed:
		ctx.Logger.Info("Deployment crashed")
		return ctx.ExecutionState.Emit(CrashedOutputChannel, DeploymentPayloadType, []any{eventData})
	default:
		// FAILED, REMOVED, SKIPPED all go to failed channel
		ctx.Logger.Infof("Deployment ended with status: %s", deployment.Status)
		return ctx.ExecutionState.Emit(FailedOutputChannel, DeploymentPayloadType, []any{eventData})
	}
}

func (c *TriggerDeploy) Cancel(ctx core.ExecutionContext) error {
	// Railway doesn't have a cancel deployment API, so we just log and return
	ctx.Logger.Info("Cancel requested - Railway deployments cannot be cancelled via API")
	return nil
}

func (c *TriggerDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *TriggerDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}
