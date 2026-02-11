package render

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
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
	deployExecutionKey         = "deploy_id"
)

type Deploy struct{}

type DeployExecutionMetadata struct {
	Deploy *DeployMetadata `json:"deploy" mapstructure:"deploy"`
}

type DeployMetadata struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	ServiceID          string `json:"serviceId"`
	CreatedAt          string `json:"createdAt"`
	FinishedAt         string `json:"finishedAt"`
	RollbackToDeployID string `json:"rollbackToDeployId,omitempty"`
}

// deployEndedWebhookConfig holds the parameters that vary between
// deploy lifecycle components (Deploy, CancelDeploy, RollbackDeploy).
type deployEndedWebhookConfig struct {
	executionKey    string
	successStatuses []string
	successChannel  string
	failedChannel   string
	payloadType     string
}

type DeployConfiguration struct {
	Service    string `json:"service" mapstructure:"service"`
	ClearCache bool   `json:"clearCache" mapstructure:"clearCache"`
}

type deployWebhookPayload struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	ServiceID string         `json:"serviceId"`
	Data      map[string]any `json:"data"`
}

type deployWebhookResult struct {
	DeployID   string
	Status     string
	ServiceID  string
	CreatedAt  string
	FinishedAt string
	EventID    string
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

func decodeDeployConfiguration(configuration any) (DeployConfiguration, error) {
	spec := DeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return DeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	if spec.Service == "" {
		return DeployConfiguration{}, fmt.Errorf("service is required")
	}

	return spec, nil
}

func (c *Deploy) Setup(ctx core.SetupContext) error {
	if _, err := decodeDeployConfiguration(ctx.Configuration); err != nil {
		return err
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
	spec, err := decodeDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.TriggerDeploy(spec.Service, spec.ClearCache)
	if err != nil {
		return err
	}

	deployID := readString(deploy.ID)
	if deployID == "" {
		return fmt.Errorf("deploy response missing id")
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

	if err := ctx.ExecutionState.SetKV(deployExecutionKey, deployID); err != nil {
		return err
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

	spec, err := decodeDeployConfiguration(ctx.Configuration)
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
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeployPollInterval)
	}

	metadata.Deploy.Status = deploy.Status
	metadata.Deploy.FinishedAt = readString(deploy.FinishedAt)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	payload := deployPayloadFromDeployResponse(deploy)
	return emitDeployStatusResult(ctx.ExecutionState, deploy.Status, deployEndedWebhookConfig{
		successStatuses: []string{"live", "succeeded"},
		successChannel:  DeploySuccessOutputChannel,
		failedChannel:   DeployFailedOutputChannel,
		payloadType:     DeployPayloadType,
	}, payload)
}

func (c *Deploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return handleDeployEndedWebhook(ctx, deployEndedWebhookConfig{
		executionKey:    deployExecutionKey,
		successStatuses: []string{"live", "succeeded"},
		successChannel:  DeploySuccessOutputChannel,
		failedChannel:   DeployFailedOutputChannel,
		payloadType:     DeployPayloadType,
	})
}

// resolveDeployFromEvent fetches deploy details from a webhook event ID.
func resolveDeployFromEvent(
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

func parseDeployWebhookPayload(body []byte) (deployWebhookPayload, error) {
	payload := deployWebhookPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return deployWebhookPayload{}, err
	}

	if payload.Data == nil {
		payload.Data = map[string]any{}
	}

	return payload, nil
}

func deployWebhookResultFromPayload(payload deployWebhookPayload) deployWebhookResult {
	serviceID := readString(payload.ServiceID)
	if serviceID == "" {
		serviceID = readString(payload.Data["serviceId"])
	}

	eventID := readString(payload.ID)
	if eventID == "" {
		eventID = readString(payload.Data["id"])
	}

	return deployWebhookResult{
		DeployID:   readString(payload.Data["deployId"]),
		Status:     readString(payload.Data["status"]),
		ServiceID:  serviceID,
		CreatedAt:  readString(payload.Data["createdAt"]),
		FinishedAt: readString(payload.Data["finishedAt"]),
		EventID:    eventID,
	}
}

func mergeDeployWebhookResults(primary, fallback deployWebhookResult) deployWebhookResult {
	if primary.DeployID == "" {
		primary.DeployID = fallback.DeployID
	}
	if primary.Status == "" {
		primary.Status = fallback.Status
	}
	if primary.ServiceID == "" {
		primary.ServiceID = fallback.ServiceID
	}
	if primary.CreatedAt == "" {
		primary.CreatedAt = fallback.CreatedAt
	}
	if primary.FinishedAt == "" {
		primary.FinishedAt = fallback.FinishedAt
	}
	if primary.EventID == "" {
		primary.EventID = fallback.EventID
	}

	return primary
}

// resolveDeployEndedWebhookResult resolves webhook payload and event data
// into a unified deployWebhookResult.
func resolveDeployEndedWebhookResult(
	ctx core.WebhookRequestContext,
	payload deployWebhookPayload,
) (deployWebhookResult, error) {
	result := deployWebhookResultFromPayload(payload)
	eventResult, err := resolveDeployFromEvent(ctx, result.EventID)
	if err != nil {
		return deployWebhookResult{}, err
	}

	return mergeDeployWebhookResults(result, eventResult), nil
}

// findExecutionByDeployID looks up an execution using a deploy ID stored in KV.
func findExecutionByDeployID(ctx core.WebhookRequestContext, key, deployID string) (*core.ExecutionContext, error) {
	if deployID == "" || ctx.FindExecutionByKV == nil {
		return nil, nil
	}

	return ctx.FindExecutionByKV(key, deployID)
}

func applyDeployWebhookResultToMetadata(metadata *DeployExecutionMetadata, result deployWebhookResult) {
	if metadata.Deploy != nil {
		if metadata.Deploy.ID == "" {
			metadata.Deploy.ID = result.DeployID
		}
		metadata.Deploy.Status = result.Status
		if result.FinishedAt != "" {
			metadata.Deploy.FinishedAt = result.FinishedAt
		}
		if metadata.Deploy.ServiceID == "" {
			metadata.Deploy.ServiceID = result.ServiceID
		}
		return
	}

	metadata.Deploy = &DeployMetadata{
		ID:         result.DeployID,
		Status:     result.Status,
		ServiceID:  result.ServiceID,
		CreatedAt:  result.CreatedAt,
		FinishedAt: result.FinishedAt,
	}
}

func deployPayloadFromDeployResponse(deploy DeployResponse) map[string]any {
	payload := map[string]any{
		"deployId":  deploy.ID,
		"status":    deploy.Status,
		"createdAt": deploy.CreatedAt,
	}
	if deploy.FinishedAt != "" {
		payload["finishedAt"] = deploy.FinishedAt
	}

	return payload
}

func deployPayloadFromWebhookResult(result deployWebhookResult) map[string]any {
	payload := map[string]any{
		"deployId":  result.DeployID,
		"status":    result.Status,
		"serviceId": result.ServiceID,
	}
	if result.EventID != "" {
		payload["eventId"] = result.EventID
	}
	if result.FinishedAt != "" {
		payload["finishedAt"] = result.FinishedAt
	}

	return payload
}

// handleDeployEndedWebhook is the shared webhook handler for deploy lifecycle
// components (Deploy, CancelDeploy, RollbackDeploy).
func handleDeployEndedWebhook(
	ctx core.WebhookRequestContext,
	config deployEndedWebhookConfig,
) (int, error) {
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

	result, err := resolveDeployEndedWebhookResult(ctx, payload)
	if err != nil {
		return http.StatusOK, nil
	}
	if result.DeployID == "" || result.Status == "" {
		return http.StatusOK, nil
	}

	executionCtx, err := findExecutionByDeployID(ctx, config.executionKey, result.DeployID)
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

	data := deployPayloadFromWebhookResult(result)
	enrichPayloadFromMetadata(data, &metadata)

	if err := emitDeployStatusResult(executionCtx.ExecutionState, readString(data["status"]), config, data); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

// emitDeployStatusResult emits to the success or failed channel based on status.
func emitDeployStatusResult(
	state core.ExecutionStateContext,
	status string,
	config deployEndedWebhookConfig,
	payload map[string]any,
) error {
	if slices.Contains(config.successStatuses, status) {
		return state.Emit(config.successChannel, config.payloadType, []any{payload})
	}
	return state.Emit(config.failedChannel, config.payloadType, []any{payload})
}

// enrichPayloadFromMetadata adds optional metadata fields (e.g. rollbackToDeployId)
// to an output payload.
func enrichPayloadFromMetadata(payload map[string]any, metadata *DeployExecutionMetadata) {
	if metadata.Deploy == nil {
		return
	}
	if metadata.Deploy.RollbackToDeployID != "" {
		payload["rollbackToDeployId"] = metadata.Deploy.RollbackToDeployID
	}
}

func (c *Deploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Deploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
