package render

import (
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
	DeployPayloadType          = "render.deploy.finished"
	DeploySuccessOutputChannel = "success"
	DeployFailedOutputChannel  = "failed"
	DeployPollInterval         = 5 * time.Minute // fallback when deploy_ended webhook doesn't arrive
)

type Deploy struct{}

type DeployExecutionMetadata struct {
	Deploy *DeployMetadata `json:"deploy" mapstructure:"deploy"`
}

type DeployMetadata struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ServiceID  string `json:"serviceId"`
	CreatedAt  string `json:"createdAt"`
	FinishedAt string `json:"finishedAt"`
}

type DeployConfiguration struct {
	Service    string `json:"service" mapstructure:"service"`
	ClearCache bool   `json:"clearCache" mapstructure:"clearCache"`
}

func (c *Deploy) Name() string {
	return "render.deploy"
}

func (c *Deploy) Label() string {
	return "Deploy"
}

func (c *Deploy) Description() string {
	return "Trigger a deploy for a Render service and wait for it to complete"
}

func (c *Deploy) Documentation() string {
	return `The Deploy component starts a new deploy for a Render service and waits for it to complete.

## Use Cases

- **Merge to deploy**: Trigger production deploys after a successful GitHub merge and CI pass
- **Scheduled redeploys**: Redeploy staging services on schedules or external content changes
- **Chained deploys**: Deploy service B when service A finishes successfully

## How It Works

1. Triggers a new deploy for the selected Render service via the Render API
2. Waits for the deploy to complete (via deploy_ended webhook and optional polling fallback)
3. Routes execution based on deploy outcome:
   - **Success channel**: Deploy completed successfully
   - **Failed channel**: Deploy failed or was cancelled

## Configuration

- **Service**: Render service to deploy
- **Clear Cache**: Clear build cache before deploying

## Output Channels

- **Success**: Emitted when the deploy completes successfully
- **Failed**: Emitted when the deploy fails or is cancelled

## Notes

- Uses the existing integration webhook for deploy_ended events (same as On Deploy trigger)
- Falls back to polling if the webhook does not arrive
- Requires a Render API key configured on the integration`
}

func (c *Deploy) Icon() string {
	return "rocket"
}

func (c *Deploy) Color() string {
	return "gray"
}

func (c *Deploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: DeploySuccessOutputChannel, Label: "Success"},
		{Name: DeployFailedOutputChannel, Label: "Failed"},
	}
}

func (c *Deploy) Configuration() []configuration.Field {
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
			Description: "Render service to deploy",
		},
		{
			Name:        "clearCache",
			Label:       "Clear Cache",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Clear build cache before triggering the deploy",
		},
	}
}

func (c *Deploy) Setup(ctx core.SetupContext) error {
	spec := DeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Service) == "" {
		return fmt.Errorf("service is required")
	}

	// Request webhook for deploy_ended so this component can receive completion events
	ctx.Integration.RequestWebhook(webhookConfigurationForResource(
		ctx.Integration,
		webhookResourceTypeDeploy,
		[]string{"deploy_ended"},
	))

	return nil
}

func (c *Deploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Deploy) Execute(ctx core.ExecutionContext) error {
	spec := DeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Service) == "" {
		return fmt.Errorf("service is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.TriggerDeploy(spec.Service, spec.ClearCache)
	if err != nil {
		return err
	}

	deployID := readString(deploy["id"])
	if deployID == "" {
		return fmt.Errorf("deploy response missing id")
	}

	status := readString(deploy["status"])
	createdAt := readString(deploy["createdAt"])
	finishedAt := readString(deploy["finishedAt"])

	err = ctx.Metadata.Set(DeployExecutionMetadata{
		Deploy: &DeployMetadata{
			ID:         deployID,
			Status:     status,
			ServiceID:  spec.Service,
			CreatedAt:  createdAt,
			FinishedAt: finishedAt,
		},
	})
	if err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV("deploy_id", deployID); err != nil {
		return err
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("Triggered deploy %s for service %s", deployID, spec.Service)
	}

	// Wait for deploy_ended webhook; poll as fallback
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeployPollInterval)
}

func (c *Deploy) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *Deploy) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *Deploy) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec := DeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	metadata := DeployExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Deploy == nil || metadata.Deploy.ID == "" {
		return nil
	}

	if isDeployFinished(metadata.Deploy.Status) {
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

	status := readString(deploy["status"])
	if !isDeployFinished(status) {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeployPollInterval)
	}

	metadata.Deploy.Status = status
	metadata.Deploy.FinishedAt = readString(deploy["finishedAt"])
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return c.emitDeployResult(ctx, deploy)
}

func (c *Deploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := verifyWebhookSignature(ctx); err != nil {
		return http.StatusForbidden, err
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	eventType := normalizeDeployWebhookEventType(readString(payload["type"]))
	if eventType != "deploy_ended" {
		return http.StatusOK, nil
	}

	data := readMap(payload["data"])
	deployID := readString(data["deployId"])
	if deployID == "" {
		deployID = readString(data["id"])
	}
	if deployID == "" {
		return http.StatusOK, nil
	}

	executionCtx, err := ctx.FindExecutionByKV("deploy_id", deployID)
	if err != nil {
		return http.StatusOK, nil
	}

	metadata := DeployExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %w", err)
	}

	if metadata.Deploy != nil && isDeployFinished(metadata.Deploy.Status) {
		return http.StatusOK, nil
	}

	status := readString(data["status"])
	if metadata.Deploy != nil {
		metadata.Deploy.Status = status
		metadata.Deploy.FinishedAt = readString(data["finishedAt"])
	} else {
		metadata.Deploy = &DeployMetadata{
			ID:         deployID,
			Status:     status,
			ServiceID:  readString(data["serviceId"]),
			CreatedAt:  readString(data["createdAt"]),
			FinishedAt: readString(data["finishedAt"]),
		}
	}
	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, err
	}

	if err := c.emitDeployResultFromWebhook(executionCtx, data); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (c *Deploy) emitDeployResult(ctx core.ActionContext, deploy map[string]any) error {
	if isDeploySucceeded(readString(deploy["status"])) {
		return ctx.ExecutionState.Emit(DeploySuccessOutputChannel, DeployPayloadType, []any{deploy})
	}
	return ctx.ExecutionState.Emit(DeployFailedOutputChannel, DeployPayloadType, []any{deploy})
}

func (c *Deploy) emitDeployResultFromWebhook(ctx *core.ExecutionContext, data map[string]any) error {
	status := readString(data["status"])
	if isDeploySucceeded(status) {
		return ctx.ExecutionState.Emit(DeploySuccessOutputChannel, DeployPayloadType, []any{data})
	}
	return ctx.ExecutionState.Emit(DeployFailedOutputChannel, DeployPayloadType, []any{data})
}

func normalizeDeployWebhookEventType(t string) string {
	t = strings.TrimSpace(strings.ToLower(t))
	if t == "render.deploy.ended" {
		return "deploy_ended"
	}
	return t
}

func isDeployFinished(status string) bool {
	s := strings.TrimSpace(strings.ToLower(status))
	return s == "succeeded" || s == "failed" || s == "canceled" || s == "cancelled"
}

func isDeploySucceeded(status string) bool {
	return strings.TrimSpace(strings.ToLower(status)) == "succeeded"
}

func (c *Deploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Deploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
