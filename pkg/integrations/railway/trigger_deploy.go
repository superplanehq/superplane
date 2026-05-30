package railway

import (
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
	TriggerDeployPayloadType          = "railway.deploy.finished"
	TriggerDeploySuccessOutputChannel = "success"
	TriggerDeployFailedOutputChannel  = "failed"
	TriggerDeployPollInterval         = 15 * time.Second
	deployExecutionKey                = "deploy_id"
)

type TriggerDeploy struct{}

type TriggerDeployConfiguration struct {
	Project     string `json:"project" mapstructure:"project"`
	Service     string `json:"service" mapstructure:"service"`
	Environment string `json:"environment" mapstructure:"environment"`
}

type TriggerDeployExecutionMetadata struct {
	Deploy *TriggerDeployMetadata `json:"deploy" mapstructure:"deploy"`
}

type TriggerDeployMetadata struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	ProjectID   string `json:"projectId"`
	ServiceID   string `json:"serviceId"`
	Environment string `json:"environmentId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (c *TriggerDeploy) Name() string {
	return "railway.triggerDeploy"
}

func (c *TriggerDeploy) Label() string {
	return "Trigger Deploy"
}

func (c *TriggerDeploy) Description() string {
	return "Trigger a deploy for a Railway service and wait for it to complete"
}

func (c *TriggerDeploy) Documentation() string {
	return `The Trigger Deploy action starts a new deploy for a Railway service and waits for it to complete.

## Configuration

- **Project**: The Railway project containing the service.
- **Service**: The service to deploy.
- **Environment**: The target environment.

## Output Channels

- **Success**: Emitted when the deploy completes successfully.
- **Failed**: Emitted when the deploy fails or is cancelled.`
}

func (c *TriggerDeploy) Icon() string {
	return "railway"
}

func (c *TriggerDeploy) Color() string {
	return "gray"
}

func (c *TriggerDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: TriggerDeploySuccessOutputChannel, Label: "Success"},
		{Name: TriggerDeployFailedOutputChannel, Label: "Failed"},
	}
}

func (c *TriggerDeploy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Railway project",
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
			Description: "The service to deploy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
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
			Description: "The target environment",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "environment",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
}

func decodeTriggerDeployConfiguration(config any) (TriggerDeployConfiguration, error) {
	spec := TriggerDeployConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return TriggerDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	spec.Service = strings.TrimSpace(spec.Service)
	spec.Environment = strings.TrimSpace(spec.Environment)

	if spec.Project == "" {
		return TriggerDeployConfiguration{}, fmt.Errorf("project is required")
	}
	if spec.Service == "" {
		return TriggerDeployConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.Environment == "" {
		return TriggerDeployConfiguration{}, fmt.Errorf("environment is required")
	}

	return spec, nil
}

func (c *TriggerDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeTriggerDeployConfiguration(ctx.Configuration)
	return err
}

func (c *TriggerDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TriggerDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeTriggerDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deployID, err := client.TriggerDeploy(spec.Environment, spec.Service)
	if err != nil {
		return err
	}

	if deployID == "" {
		return fmt.Errorf("deploy response missing deployment ID")
	}

	err = ctx.Metadata.Set(TriggerDeployExecutionMetadata{
		Deploy: &TriggerDeployMetadata{
			ID:          deployID,
			Status:      "QUEUED",
			ProjectID:   spec.Project,
			ServiceID:   spec.Service,
			Environment: spec.Environment,
		},
	})
	if err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(deployExecutionKey, deployID); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, TriggerDeployPollInterval)
}

func (c *TriggerDeploy) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: "poll",
			Type: core.HookTypeInternal,
		},
	}
}

func (c *TriggerDeploy) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *TriggerDeploy) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := TriggerDeployExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Deploy == nil || metadata.Deploy.ID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.GetDeployment(metadata.Deploy.ID)
	if err != nil {
		return err
	}

	metadata.Deploy.Status = deploy.Status
	metadata.Deploy.CreatedAt = deploy.CreatedAt
	metadata.Deploy.UpdatedAt = deploy.UpdatedAt
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	// Terminal states check
	payload := map[string]any{
		"deployId":      deploy.ID,
		"status":        deploy.Status,
		"projectId":     metadata.Deploy.ProjectID,
		"serviceId":     metadata.Deploy.ServiceID,
		"environmentId": metadata.Deploy.Environment,
	}

	switch deploy.Status {
	case "SUCCESS":
		return ctx.ExecutionState.Emit(TriggerDeploySuccessOutputChannel, TriggerDeployPayloadType, []any{payload})
	case "FAILED", "CRASHED", "REMOVED", "SKIPPED", "SLEEPING":
		return ctx.ExecutionState.Emit(TriggerDeployFailedOutputChannel, TriggerDeployPayloadType, []any{payload})
	default:
		// Not finished yet, schedule next poll
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, TriggerDeployPollInterval)
	}
}

func (c *TriggerDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *TriggerDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TriggerDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *TriggerDeploy) ExampleOutput() map[string]any {
	return map[string]any{
		"deployId":      "ebda9796-09e4-456f-af60-d1a66dee66a0",
		"status":        "SUCCESS",
		"projectId":     "8db400fa-357e-4646-90f0-c7eb36e88a92",
		"serviceId":     "2a345678-bcde-4fgh-1234-567812345678",
		"environmentId": "9a1d7a89-2cf4-4446-9b69-4cde850918aa",
	}
}
