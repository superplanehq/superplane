package render

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
	RollbackDeployPayloadType          = "render.deploy.finished"
	RollbackDeploySuccessOutputChannel = "success"
	RollbackDeployFailedOutputChannel  = "failed"
	RollbackDeployPollInterval         = 5 * time.Minute
	rollbackDeployExecutionKey         = "deploy_id"
)

type RollbackDeploy struct{}

type RollbackDeployConfiguration struct {
	Service            string `json:"service" mapstructure:"service"`
	RollbackToDeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *RollbackDeploy) Name() string {
	return "render.rollbackDeploy"
}

func (c *RollbackDeploy) Label() string {
	return "Rollback Deploy"
}

func (c *RollbackDeploy) Description() string {
	return "Roll back a Render service to a previous deploy and wait for it to complete"
}

func (c *RollbackDeploy) Documentation() string {
	return `The Rollback Deploy component triggers a rollback deploy for a Render service and waits for it to complete.

## Use Cases

- **Automated recovery**: Roll back after detecting errors in a new deploy
- **One-click rollback**: Trigger rollbacks from an incident workflow

## How It Works

1. Triggers a rollback deploy for the selected Render service via the Render API
2. Waits for the deploy to complete (via deploy_ended webhook and optional polling fallback)
3. Routes execution based on deploy outcome:
   - **Success channel**: Deploy completed successfully (status is ` + "`live`" + `)
   - **Failed channel**: Deploy failed or was cancelled

## Configuration

- **Service**: Render service to roll back
- **Deploy ID**: The deploy ID to roll back to (supports expressions)

## Output Channels

- **Success**: Emitted when the rollback deploy completes successfully
- **Failed**: Emitted when the rollback deploy fails or is cancelled

## Notes

- Uses the existing integration webhook for deploy_ended events
- Falls back to polling if the webhook does not arrive
- Includes ` + "`rollbackToDeployId`" + ` in the output payload for reference
- Requires a Render API key configured on the integration`
}

func (c *RollbackDeploy) Icon() string {
	return "rotate-ccw"
}

func (c *RollbackDeploy) Color() string {
	return "gray"
}

func (c *RollbackDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: RollbackDeploySuccessOutputChannel, Label: "Success"},
		{Name: RollbackDeployFailedOutputChannel, Label: "Failed"},
	}
}

func (c *RollbackDeploy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service to roll back",
		},
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dep-... or {{$.event.data.deployId}}",
			Description: "Deploy ID to roll back to",
		},
	}
}

func decodeRollbackDeployConfiguration(configuration any) (RollbackDeployConfiguration, error) {
	spec := RollbackDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return RollbackDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.RollbackToDeployID = strings.TrimSpace(spec.RollbackToDeployID)
	if spec.Service == "" {
		return RollbackDeployConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.RollbackToDeployID == "" {
		return RollbackDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *RollbackDeploy) Setup(ctx core.SetupContext) error {
	if _, err := decodeRollbackDeployConfiguration(ctx.Configuration); err != nil {
		return err
	}

	ctx.Integration.RequestWebhook(webhookConfigurationForResource(
		ctx.Integration,
		webhookResourceTypeDeploy,
		[]string{"deploy_ended"},
	))

	return nil
}

func (c *RollbackDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RollbackDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRollbackDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.RollbackDeploy(spec.Service, spec.RollbackToDeployID)
	if err != nil {
		return err
	}

	deployID := readString(deploy.ID)
	if deployID == "" {
		return fmt.Errorf("rollback deploy response missing id")
	}

	err = ctx.Metadata.Set(DeployExecutionMetadata{
		Deploy: &DeployMetadata{
			ID:         deployID,
			Status:     readString(deploy.Status),
			ServiceID:  spec.Service,
			CreatedAt:  readString(deploy.CreatedAt),
			FinishedAt: readString(deploy.FinishedAt),
		},
	})
	if err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(rollbackDeployExecutionKey, deployID); err != nil {
		return err
	}

	// Wait for deploy_ended webhook; poll as fallback
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, RollbackDeployPollInterval)
}

func (c *RollbackDeploy) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *RollbackDeploy) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *RollbackDeploy) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec, err := decodeRollbackDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := DeployExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Deploy == nil || metadata.Deploy.ID == "" {
		return nil
	}

	if metadata.Deploy.FinishedAt != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.GetDeploy(spec.Service, metadata.Deploy.ID)
	if err != nil {
		return err
	}

	if deploy.FinishedAt == "" {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, RollbackDeployPollInterval)
	}

	metadata.Deploy.Status = deploy.Status
	metadata.Deploy.FinishedAt = readString(deploy.FinishedAt)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return c.emitPollResult(ctx, deploy)
}

func (c *RollbackDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := verifyWebhookSignature(ctx); err != nil {
		return http.StatusForbidden, err
	}

	payload, err := parseDeployWebhookPayload(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	if readString(payload.Type) != "deploy_ended" {
		return http.StatusOK, nil
	}

	result, err := c.resolveWebhookResult(ctx, payload)
	if err != nil {
		return http.StatusOK, nil
	}
	if result.DeployID == "" || result.Status == "" {
		return http.StatusOK, nil
	}

	executionCtx, err := findRollbackDeployExecutionByID(ctx, result.DeployID)
	if err != nil {
		return http.StatusOK, nil
	}
	if executionCtx == nil {
		return http.StatusOK, nil
	}

	metadata := DeployExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %w", err)
	}

	if metadata.Deploy != nil && metadata.Deploy.FinishedAt != "" {
		return http.StatusOK, nil
	}

	applyDeployWebhookResultToMetadata(&metadata, result)
	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, err
	}

	if err := c.emitWebhookResult(executionCtx, deployPayloadFromWebhookResult(result)); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (c *RollbackDeploy) resolveWebhookResult(
	ctx core.WebhookRequestContext,
	payload deployWebhookPayload,
) (deployWebhookResult, error) {
	result := deployWebhookResultFromPayload(payload)
	eventResult, err := c.resolveDeployFromEvent(ctx, result.EventID)
	if err != nil {
		return deployWebhookResult{}, err
	}

	return mergeDeployWebhookResults(result, eventResult), nil
}

func (c *RollbackDeploy) resolveDeployFromEvent(
	ctx core.WebhookRequestContext,
	eventID string,
) (deployWebhookResult, error) {
	if eventID == "" {
		return deployWebhookResult{}, nil
	}

	event, err := resolveWebhookEvent(ctx, eventID)
	if err != nil {
		return deployWebhookResult{}, err
	}

	detailValues := eventDetailValues(event)
	if detailValues.DeployID == "" {
		return deployWebhookResult{}, nil
	}

	return deployWebhookResult{
		DeployID:   detailValues.DeployID,
		ServiceID:  readString(event.ServiceID),
		FinishedAt: readString(event.Timestamp),
		EventID:    eventID,
	}, nil
}

func findRollbackDeployExecutionByID(ctx core.WebhookRequestContext, deployID string) (*core.ExecutionContext, error) {
	if deployID == "" || ctx.FindExecutionByKV == nil {
		return nil, nil
	}

	return ctx.FindExecutionByKV(rollbackDeployExecutionKey, deployID)
}

func (c *RollbackDeploy) emitPollResult(ctx core.ActionContext, deploy DeployResponse) error {
	payload := deployPayloadFromDeployResponse(deploy)
	if deploy.Status == "live" {
		return ctx.ExecutionState.Emit(RollbackDeploySuccessOutputChannel, RollbackDeployPayloadType, []any{payload})
	}
	return ctx.ExecutionState.Emit(RollbackDeployFailedOutputChannel, RollbackDeployPayloadType, []any{payload})
}

func (c *RollbackDeploy) emitWebhookResult(ctx *core.ExecutionContext, data map[string]any) error {
	status := readString(data["status"])
	if status == "live" {
		return ctx.ExecutionState.Emit(RollbackDeploySuccessOutputChannel, RollbackDeployPayloadType, []any{data})
	}
	return ctx.ExecutionState.Emit(RollbackDeployFailedOutputChannel, RollbackDeployPayloadType, []any{data})
}

func (c *RollbackDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RollbackDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
