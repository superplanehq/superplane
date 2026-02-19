package terraformcloud

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const TriggerRunPayloadType = "terraformcloud.run"
const TriggerRunSuccessChannel = "success"
const TriggerRunFailedChannel = "failed"
const TriggerRunPollInterval = 5 * time.Minute

type TriggerRun struct{}

type TriggerRunNodeMetadata struct {
	WorkspaceID   string `json:"workspaceId" mapstructure:"workspaceId"`
	WorkspaceName string `json:"workspaceName" mapstructure:"workspaceName"`
	Organization  string `json:"organization" mapstructure:"organization"`
}

type TriggerRunExecutionMetadata struct {
	Run RunInfo `json:"run" mapstructure:"run"`
}

type RunInfo struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	RunURL    string `json:"run_url"`
}

type TriggerRunSpec struct {
	Organization string `json:"organization"`
	WorkspaceID  string `json:"workspaceId"`
	Message      string `json:"message"`
}

func (t *TriggerRun) Name() string {
	return "terraformcloud.triggerRun"
}

func (t *TriggerRun) Label() string {
	return "Trigger Run"
}

func (t *TriggerRun) Description() string {
	return "Trigger a Terraform Cloud run and wait for completion"
}

func (t *TriggerRun) Documentation() string {
	return `The Trigger Run component creates a new run in a Terraform Cloud workspace and waits for it to complete.

## Use Cases

- **Infrastructure deployment**: Trigger Terraform runs from SuperPlane workflows
- **Scheduled applies**: Run Terraform on a schedule or in response to events
- **Approval-gated deployments**: Combine with approval components for controlled infrastructure changes
- **Multi-workspace orchestration**: Coordinate runs across multiple workspaces

## How It Works

1. Creates a new Terraform Cloud run in the specified workspace
2. Waits for the run to reach a terminal state (monitored via webhook and polling)
3. Routes execution based on run results:
   - **Success channel**: Run completed successfully (applied or planned_and_finished)
   - **Failed channel**: Run failed, was cancelled, or errored

## Configuration

- **Organization**: Terraform Cloud organization name
- **Workspace**: The workspace to trigger the run in
- **Message**: Optional message for the run

## Output Channels

- **Success**: Emitted when the run completes successfully
- **Failed**: Emitted when the run fails, errors, or is cancelled
`
}

func (t *TriggerRun) Icon() string {
	return "cloud"
}

func (t *TriggerRun) Color() string {
	return "purple"
}

func (t *TriggerRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  TriggerRunSuccessChannel,
			Label: "Success",
		},
		{
			Name:  TriggerRunFailedChannel,
			Label: "Failed",
		},
	}
}

func (t *TriggerRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Terraform Cloud organization name",
			Placeholder: "my-organization",
		},
		{
			Name:        "workspaceId",
			Label:       "Workspace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The workspace to trigger the run in",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeWorkspace,
					Parameters: []configuration.ParameterRef{
						{
							Name: "organization",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "organization",
							},
						},
					},
				},
			},
		},
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeString,
			Description: "Optional message for the run",
			Placeholder: "Triggered by SuperPlane",
			Default:     "Triggered by SuperPlane",
		},
	}
}

func (t *TriggerRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *TriggerRun) Setup(ctx core.SetupContext) error {
	config := TriggerRunSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Organization == "" {
		return fmt.Errorf("organization is required")
	}

	if config.WorkspaceID == "" {
		return fmt.Errorf("workspace is required")
	}

	metadata := TriggerRunNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	workspaceChanged := metadata.WorkspaceID != config.WorkspaceID ||
		metadata.Organization != config.Organization

	if workspaceChanged {
		workspace, err := client.GetWorkspace(config.WorkspaceID)
		if err != nil {
			return fmt.Errorf("workspace not found or inaccessible: %w", err)
		}

		err = ctx.Metadata.Set(TriggerRunNodeMetadata{
			WorkspaceID:   workspace.ID,
			WorkspaceName: workspace.Attributes.Name,
			Organization:  config.Organization,
		})
		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WorkspaceID: config.WorkspaceID,
		Triggers:    []string{"run:completed", "run:errored"},
	})
}

func (t *TriggerRun) Execute(ctx core.ExecutionContext) error {
	spec := TriggerRunSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	run, err := client.CreateRun(spec.WorkspaceID, spec.Message, false)
	if err != nil {
		return fmt.Errorf("failed to create run: %w", err)
	}

	hostname := defaultHostname
	hostnameBytes, hostnameErr := ctx.Integration.GetConfig("hostname")
	if hostnameErr == nil && len(hostnameBytes) > 0 {
		hostname = string(hostnameBytes)
	}

	metadata := TriggerRunExecutionMetadata{
		Run: RunInfo{
			ID:        run.ID,
			Status:    run.Attributes.Status,
			Message:   run.Attributes.Message,
			CreatedAt: run.Attributes.CreatedAt,
			RunURL:    fmt.Sprintf("https://%s/app/runs/%s", hostname, run.ID),
		},
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	err = ctx.ExecutionState.SetKV("run", run.ID)
	if err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, TriggerRunPollInterval)
}

func (t *TriggerRun) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *TriggerRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Tfe-Notification-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	h := hmac.New(sha512.New, secret)
	h.Write(ctx.Body)
	computed := fmt.Sprintf("%x", h.Sum(nil))
	if computed != signature {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	runID, _ := data["run_id"].(string)
	if runID == "" {
		return http.StatusBadRequest, fmt.Errorf("run_id missing from webhook payload")
	}

	executionCtx, err := ctx.FindExecutionByKV("run", runID)
	if err != nil {
		return http.StatusOK, nil
	}

	notifications, ok := data["notifications"].([]any)
	if !ok || len(notifications) == 0 {
		return http.StatusOK, nil
	}

	firstNotification, ok := notifications[0].(map[string]any)
	if !ok {
		return http.StatusOK, nil
	}

	runStatus, _ := firstNotification["run_status"].(string)

	if !IsTerminalRunStatus(runStatus) {
		return http.StatusOK, nil
	}

	var execMetadata TriggerRunExecutionMetadata
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &execMetadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	execMetadata.Run.Status = runStatus
	err = executionCtx.Metadata.Set(execMetadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update metadata: %w", err)
	}

	payload := map[string]any{"run": execMetadata.Run}
	channel := TriggerRunSuccessChannel
	if !IsSuccessRunStatus(runStatus) {
		channel = TriggerRunFailedChannel
	}

	err = executionCtx.ExecutionState.Emit(channel, TriggerRunPayloadType, []any{payload})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit output: %w", err)
	}

	return http.StatusOK, nil
}

func (t *TriggerRun) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (t *TriggerRun) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return t.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *TriggerRun) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := TriggerRunExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Run.ID == "" {
		return fmt.Errorf("run ID is missing from execution metadata")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	run, err := client.GetRun(metadata.Run.ID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, TriggerRunPollInterval)
	}

	if !IsTerminalRunStatus(run.Attributes.Status) {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, TriggerRunPollInterval)
	}

	metadata.Run.Status = run.Attributes.Status
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	payload := map[string]any{"run": metadata.Run}
	channel := TriggerRunSuccessChannel
	if !IsSuccessRunStatus(run.Attributes.Status) {
		channel = TriggerRunFailedChannel
	}

	return ctx.ExecutionState.Emit(channel, TriggerRunPayloadType, []any{payload})
}

func (t *TriggerRun) Cleanup(ctx core.SetupContext) error {
	return nil
}
