package railway

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// TriggerDeploy is a stub for now - will be fully implemented in the next phase
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
	return `The Trigger Deploy component starts a new deployment for a Railway service in a specific environment.

## Use Cases

- **Deploy on merge**: Automatically deploy when code is merged to main
- **Scheduled deployments**: Deploy on a schedule (e.g., nightly releases)
- **Manual approval**: Deploy after approval in the workflow
- **Cross-service orchestration**: Deploy services in sequence

## Configuration

- **Project**: Select the Railway project
- **Service**: Select the service to deploy
- **Environment**: Select the target environment (e.g., production, staging)

## How It Works

1. Calls Railway's ` + "`environmentTriggersDeploy`" + ` API
2. Railway queues a new deployment for the service
3. Component emits the deployment trigger result

## Output

The component emits:
- ` + "`project`" + `: Project ID
- ` + "`service`" + `: Service ID
- ` + "`environment`" + `: Environment ID
- ` + "`triggered`" + `: Whether the deployment was triggered`
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
		core.DefaultOutputChannel,
	}
}

func (c *TriggerDeploy) ExampleOutput() map[string]any {
	return map[string]any{
		"project":     "proj-xyz789",
		"service":     "srv-ghi012",
		"environment": "env-def456",
		"triggered":   true,
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

	// Call environmentTriggersDeploy mutation
	if err := client.TriggerDeploy(config.Service, config.Environment); err != nil {
		return fmt.Errorf("failed to trigger deployment: %w", err)
	}

	ctx.Logger.Infof(
		"Triggered deployment for service %s in environment %s",
		config.Service,
		config.Environment,
	)

	// Emit result
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"railway.deployment.triggered",
		[]any{map[string]any{
			"project":     config.Project,
			"service":     config.Service,
			"environment": config.Environment,
			"triggered":   true,
		}},
	)
}

func (c *TriggerDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TriggerDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *TriggerDeploy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *TriggerDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TriggerDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *TriggerDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}
