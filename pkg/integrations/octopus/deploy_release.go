package octopus

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
	DeployReleasePayloadType          = "octopus.deployment.finished"
	DeployReleaseSuccessOutputChannel = "success"
	DeployReleaseFailedOutputChannel  = "failed"
	DeployReleasePollInterval         = 5 * time.Minute
	deployReleaseExecutionKey         = "deployment_id"
)

type DeployRelease struct{}

type DeployReleaseConfiguration struct {
	Project     string `json:"project" mapstructure:"project"`
	Release     string `json:"release" mapstructure:"release"`
	Environment string `json:"environment" mapstructure:"environment"`
}

type DeployReleaseExecutionMetadata struct {
	Deployment *DeploymentMetadata `json:"deployment" mapstructure:"deployment"`
}

type DeploymentMetadata struct {
	ID            string `json:"id"`
	TaskID        string `json:"taskId"`
	TaskState     string `json:"taskState"`
	ProjectID     string `json:"projectId"`
	ReleaseID     string `json:"releaseId"`
	EnvironmentID string `json:"environmentId"`
	Created       string `json:"created"`
	CompletedTime string `json:"completedTime,omitempty"`
}

func (c *DeployRelease) Name() string {
	return "octopus.deployRelease"
}

func (c *DeployRelease) Label() string {
	return "Deploy Release"
}

func (c *DeployRelease) Description() string {
	return "Deploy a release to an environment in Octopus Deploy"
}

func (c *DeployRelease) Documentation() string {
	return `The Deploy Release component deploys a chosen release to a chosen environment in Octopus Deploy and waits for completion.

## Use Cases

- **Deploy on merge**: Trigger a deployment after code is merged
- **Scheduled deployments**: Deploy to staging or production on a schedule
- **Approval-based deploys**: Deploy after manual approval in a workflow
- **Chained deployments**: Deploy to next environment after success in the previous one

## How It Works

1. Creates a deployment for the selected release and environment via the Octopus Deploy API
2. Waits for the deployment task to complete (via webhook and polling fallback)
3. Routes execution based on deployment outcome:
   - **Success channel**: Task completed successfully
   - **Failed channel**: Task failed, timed out, or was cancelled

## Configuration

- **Project**: The Octopus Deploy project
- **Release**: The release to deploy (filtered by the selected project)
- **Environment**: The target deployment environment

## Output Channels

- **Success**: Emitted when the deployment completes successfully
- **Failed**: Emitted when the deployment fails, times out, or is cancelled

## Notes

- Deployment status is tracked via the Octopus Deploy task associated with the deployment
- Polls the task status every 5 minutes as a fallback if the webhook does not arrive
- Requires an API key configured on the Octopus Deploy integration`
}

func (c *DeployRelease) Icon() string {
	return "rocket"
}

func (c *DeployRelease) Color() string {
	return "blue"
}

func (c *DeployRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: DeployReleaseSuccessOutputChannel, Label: "Success"},
		{Name: DeployReleaseFailedOutputChannel, Label: "Failed"},
	}
}

func (c *DeployRelease) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Octopus Deploy project",
		},
		{
			Name:     "release",
			Label:    "Release",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "release",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
			Description: "Release to deploy",
		},
		{
			Name:     "environment",
			Label:    "Environment",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "environment",
				},
			},
			Description: "Target deployment environment",
		},
	}
}

func decodeDeployReleaseConfiguration(configuration any) (DeployReleaseConfiguration, error) {
	spec := DeployReleaseConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return DeployReleaseConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	if spec.Project == "" {
		return DeployReleaseConfiguration{}, fmt.Errorf("project is required")
	}

	spec.Release = strings.TrimSpace(spec.Release)
	if spec.Release == "" {
		return DeployReleaseConfiguration{}, fmt.Errorf("release is required")
	}

	spec.Environment = strings.TrimSpace(spec.Environment)
	if spec.Environment == "" {
		return DeployReleaseConfiguration{}, fmt.Errorf("environment is required")
	}

	return spec, nil
}

func (c *DeployRelease) Setup(ctx core.SetupContext) error {
	spec, err := decodeDeployReleaseConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	// Resolve human-readable names for display in the UI
	nodeMetadata := resolveNodeMetadata(ctx.HTTP, ctx.Integration, spec.Project, spec.Release, spec.Environment)
	if err := ctx.Metadata.Set(nodeMetadata); err != nil {
		return fmt.Errorf("failed to store node metadata: %w", err)
	}

	// Request webhook for deployment events so this component can receive completion notifications
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventCategories: []string{
			EventCategoryDeploymentSucceeded,
			EventCategoryDeploymentFailed,
		},
	})
}

func (c *DeployRelease) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeployRelease) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeployReleaseConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	spaceID, err := spaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return err
	}

	deployment, err := client.CreateDeployment(spaceID, spec.Release, spec.Environment)
	if err != nil {
		return err
	}

	if deployment.ID == "" {
		return fmt.Errorf("deployment response missing ID")
	}

	if deployment.TaskID == "" {
		return fmt.Errorf("deployment response missing task ID")
	}

	err = ctx.Metadata.Set(DeployReleaseExecutionMetadata{
		Deployment: &DeploymentMetadata{
			ID:            deployment.ID,
			TaskID:        deployment.TaskID,
			TaskState:     TaskStateQueued,
			ProjectID:     deployment.ProjectID,
			ReleaseID:     deployment.ReleaseID,
			EnvironmentID: deployment.EnvironmentID,
			Created:       deployment.Created,
		},
	})
	if err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(deployReleaseExecutionKey, deployment.ID); err != nil {
		return err
	}

	// Wait for webhook; poll as fallback
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeployReleasePollInterval)
}

func (c *DeployRelease) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *DeployRelease) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *DeployRelease) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := DeployReleaseExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Deployment == nil || metadata.Deployment.TaskID == "" {
		return nil
	}

	if metadata.Deployment.CompletedTime != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	task, err := client.GetTask(metadata.Deployment.TaskID)
	if err != nil {
		return err
	}

	if !isTaskCompleted(task.State) {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, DeployReleasePollInterval)
	}

	metadata.Deployment.TaskState = task.State
	metadata.Deployment.CompletedTime = task.CompletedTime
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	payload := buildDeployReleasePayload(metadata.Deployment, task)
	return emitDeployResult(ctx.ExecutionState, task.State, payload)
}

func (c *DeployRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := verifyWebhookHeader(ctx); err != nil {
		return http.StatusForbidden, err
	}

	if !webhookRequestIsJSON(ctx) {
		return okResponse()
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return errorResponse(http.StatusBadRequest, "error parsing request body: %w", err)
	}

	// The actual event category is in Payload.Event.Category, not the top-level EventType
	// (EventType is always "SubscriptionPayload").
	eventPayload := readMap(payload["Payload"])
	event := readMap(eventPayload["Event"])
	eventType := readString(event["Category"])
	if eventType != EventCategoryDeploymentSucceeded && eventType != EventCategoryDeploymentFailed {
		return okResponse()
	}

	// Try to find a matching execution from all deployment IDs in the event payload.
	// Octopus may include multiple deployment IDs (e.g. multi-tenant, chained deployments).
	relatedDocs := readRelatedDocumentIDs(event)
	deploymentIDs := relatedDocs["Deployments"]

	if len(deploymentIDs) == 0 {
		return okResponse()
	}

	var executionCtx *core.ExecutionContext
	for _, id := range deploymentIDs {
		found, err := findExecutionByDeploymentID(ctx, id)
		if err != nil {
			continue
		}
		if found != nil {
			executionCtx = found
			break
		}
	}

	if executionCtx == nil {
		return okResponse()
	}

	if executionCtx.ExecutionState.IsFinished() {
		return okResponse()
	}

	metadata := DeployReleaseExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return errorResponse(http.StatusInternalServerError, "error decoding metadata: %w", err)
	}

	if metadata.Deployment != nil && metadata.Deployment.CompletedTime != "" {
		return okResponse()
	}

	resolvedTask := Task{
		CompletedTime: readString(payload["Timestamp"]),
	}

	// Default state from event type; used when client creation or GetTask fails.
	if eventType == EventCategoryDeploymentSucceeded {
		resolvedTask.State = TaskStateSuccess
	} else {
		resolvedTask.State = TaskStateFailed
	}

	// Try to enrich with real task data; fall back gracefully on client/API errors.
	// Only replace the event-derived state when the API confirms the task is completed;
	// a non-completed state (e.g. Executing) indicates a race condition or caching artefact
	// and must not overwrite the webhook-event-based state.
	if metadata.Deployment != nil && metadata.Deployment.TaskID != "" {
		if client, err := NewClient(ctx.HTTP, ctx.Integration); err == nil {
			if task, taskErr := client.GetTask(metadata.Deployment.TaskID); taskErr == nil && isTaskCompleted(task.State) {
				// Prefer the real task state (may be Canceled, TimedOut, etc.)
				resolvedTask = task
				if resolvedTask.CompletedTime == "" {
					resolvedTask.CompletedTime = readString(payload["Timestamp"])
				}
			}
		}
	}

	if metadata.Deployment == nil {
		metadata.Deployment = &DeploymentMetadata{}
	}

	metadata.Deployment.TaskState = resolvedTask.State
	metadata.Deployment.CompletedTime = resolvedTask.CompletedTime
	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return errorResponse(http.StatusInternalServerError, "error updating metadata: %w", err)
	}

	deployPayload := buildDeployReleasePayload(metadata.Deployment, resolvedTask)

	if err := emitDeployResult(executionCtx.ExecutionState, resolvedTask.State, deployPayload); err != nil {
		return errorResponse(http.StatusInternalServerError, "error emitting result: %w", err)
	}

	return okResponse()
}

func findExecutionByDeploymentID(ctx core.WebhookRequestContext, deploymentID string) (*core.ExecutionContext, error) {
	if deploymentID == "" || ctx.FindExecutionByKV == nil {
		return nil, nil
	}

	return ctx.FindExecutionByKV(deployReleaseExecutionKey, deploymentID)
}

func emitDeployResult(state core.ExecutionStateContext, taskState string, payload map[string]any) error {
	if isTaskSuccessful(taskState) {
		return state.Emit(DeployReleaseSuccessOutputChannel, DeployReleasePayloadType, []any{payload})
	}
	return state.Emit(DeployReleaseFailedOutputChannel, DeployReleasePayloadType, []any{payload})
}

func buildDeployReleasePayload(deployment *DeploymentMetadata, task Task) map[string]any {
	payload := map[string]any{
		"deploymentId":  deployment.ID,
		"taskState":     task.State,
		"projectId":     deployment.ProjectID,
		"releaseId":     deployment.ReleaseID,
		"environmentId": deployment.EnvironmentID,
		"created":       deployment.Created,
	}

	if task.CompletedTime != "" {
		payload["completedTime"] = task.CompletedTime
	}

	if task.ErrorMessage != "" {
		payload["errorMessage"] = task.ErrorMessage
	}

	if task.Duration != "" {
		payload["duration"] = task.Duration
	}

	return payload
}

func (c *DeployRelease) Cancel(ctx core.ExecutionContext) error {
	metadata := DeployReleaseExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}

	if metadata.Deployment == nil || metadata.Deployment.TaskID == "" {
		return nil
	}

	if isTaskCompleted(metadata.Deployment.TaskState) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	spaceID, err := spaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil
	}

	_ = client.CancelTask(spaceID, metadata.Deployment.TaskID)
	return nil
}

func (c *DeployRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}
