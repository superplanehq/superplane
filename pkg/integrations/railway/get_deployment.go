package railway

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetDeploymentPayloadType = "railway.deployment"

type GetDeployment struct{}

type GetDeploymentConfiguration struct {
	DeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *GetDeployment) Name() string {
	return "railway.getDeployment"
}

func (c *GetDeployment) Label() string {
	return "Get Deployment"
}

func (c *GetDeployment) Description() string {
	return "Retrieve a Railway deployment by ID"
}

func (c *GetDeployment) Documentation() string {
	return `The Get Deployment action retrieves the current details of a Railway deployment.

## Configuration

- **Deploy ID**: The Railway deployment ID to retrieve.`
}

func (c *GetDeployment) Icon() string {
	return "railway"
}

func (c *GetDeployment) Color() string {
	return "gray"
}

func (c *GetDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDeployment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., {{$['Trigger Deploy'].data.deployId}}",
			Description: "Railway deployment ID to retrieve",
		},
	}
}

func decodeGetDeploymentConfiguration(configuration any) (GetDeploymentConfiguration, error) {
	spec := GetDeploymentConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetDeploymentConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.DeployID = strings.TrimSpace(spec.DeployID)
	if spec.DeployID == "" {
		return GetDeploymentConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *GetDeployment) Setup(ctx core.SetupContext) error {
	_, err := decodeGetDeploymentConfiguration(ctx.Configuration)
	return err
}

func (c *GetDeployment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDeployment) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetDeploymentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deployment, err := client.GetDeployment(spec.DeployID)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetDeploymentPayloadType, []any{deploymentData(deployment)})
}

func (c *GetDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *GetDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetDeployment) ExampleOutput() map[string]any {
	return map[string]any{
		"deployId":      "ebda9796-09e4-456f-af60-d1a66dee66a0",
		"status":        "SUCCESS",
		"projectId":     "8db400fa-357e-4646-90f0-c7eb36e88a92",
		"serviceId":     "2a345678-bcde-4fgh-1234-567812345678",
		"environmentId": "9a1d7a89-2cf4-4446-9b69-4cde850918aa",
		"canRollback":   true,
	}
}

func deploymentData(deployment *Deployment) map[string]any {
	return map[string]any{
		"deployId":          deployment.ID,
		"status":            deployment.Status,
		"createdAt":         deployment.CreatedAt,
		"updatedAt":         deployment.UpdatedAt,
		"statusUpdatedAt":   deployment.StatusUpdatedAt,
		"projectId":         deployment.ProjectID,
		"serviceId":         deployment.ServiceID,
		"environmentId":     deployment.EnvironmentID,
		"snapshotId":        deployment.SnapshotID,
		"staticUrl":         deployment.StaticURL,
		"url":               deployment.URL,
		"canRollback":       deployment.CanRollback,
		"canRedeploy":       deployment.CanRedeploy,
		"deploymentStopped": deployment.DeploymentStopped,
		"meta":              deployment.Meta,
		"diagnosis":         deployment.Diagnosis,
		"creator":           deployment.Creator,
	}
}
