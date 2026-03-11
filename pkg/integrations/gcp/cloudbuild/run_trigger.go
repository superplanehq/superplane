package cloudbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RunTrigger struct{}

type RunTriggerConfiguration struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
	TriggerID string `json:"trigger" mapstructure:"trigger"`
	Ref       string `json:"ref" mapstructure:"ref"`
}

type RunTriggerNodeMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId,omitempty"`
	TriggerName    string `json:"triggerName,omitempty" mapstructure:"triggerName,omitempty"`
}

func (c *RunTrigger) Name() string {
	return "gcp.cloudbuild.runTrigger"
}

func (c *RunTrigger) Label() string {
	return "Cloud Build • Run Trigger"
}

func (c *RunTrigger) Description() string {
	return "Run a Cloud Build trigger and wait for the build to finish"
}

func (c *RunTrigger) Documentation() string {
	return `Runs an existing Cloud Build trigger and waits for the resulting build to reach a terminal status.

## Configuration

- **Trigger** (required): The Cloud Build trigger to run. Select from triggers in the connected project.
- **Branch or tag**: Override the branch or tag to build from. Leave empty to use the trigger's configured default. A 40-character hex string is treated as a commit SHA.
- **Project ID Override**: Optionally run the trigger in a different project than the connected integration.

## Output

The terminal Build resource, including ` + "`id`" + `, ` + "`status`" + `, ` + "`logUrl`" + `, ` + "`createTime`" + `, ` + "`finishTime`" + `, and more.

## Output Channels

- **Passed**: Emitted when Cloud Build finishes with ` + "`SUCCESS`" + `.
- **Failed**: Emitted when Cloud Build finishes with any other terminal status, including ` + "`FAILURE`" + `, ` + "`INTERNAL_ERROR`" + `, ` + "`TIMEOUT`" + `, ` + "`CANCELLED`" + `, or ` + "`EXPIRED`" + `.

## Notes

- SuperPlane listens for Cloud Build notifications through the connected GCP integration and falls back to polling if an event does not arrive.
- SuperPlane automatically creates the shared ` + "`cloud-builds`" + ` Pub/Sub topic and push subscription when the GCP integration has ` + "`roles/pubsub.admin`" + ` and both the **Cloud Build** and **Pub/Sub** APIs are enabled.
- Cancelling the running execution from the UI sends a Cloud Build cancel request for the active build.`
}

func (c *RunTrigger) Icon() string  { return "gcp" }
func (c *RunTrigger) Color() string { return "gray" }

func (c *RunTrigger) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: createBuildPassedOutputChannel, Label: "Passed"},
		{Name: createBuildFailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunTrigger) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "trigger",
			Label:       "Trigger",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Cloud Build trigger to run.",
			Placeholder: "Select a trigger",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeTrigger,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "ref",
			Label:       "Branch or tag",
			Type:        configuration.FieldTypeGitRef,
			Required:    false,
			Description: "Override the branch or tag to build from. Leave empty to use the trigger's configured default.",
			Default:     "main",
		},
	}
}

func decodeRunTriggerConfiguration(raw any) (RunTriggerConfiguration, error) {
	var config RunTriggerConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return RunTriggerConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.TriggerID = strings.TrimSpace(config.TriggerID)
	config.Ref = strings.TrimSpace(config.Ref)
	return config, nil
}

func fetchTriggerName(ctx core.SetupContext, config RunTriggerConfiguration) string {
	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ""
	}
	projectID := config.ProjectID
	if projectID == "" {
		projectID = client.ProjectID()
	}
	if projectID == "" {
		return ""
	}
	data, err := client.GetURL(context.Background(), buildGetTriggerURL(projectID, config.TriggerID))
	if err != nil {
		return ""
	}
	var trigger struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &trigger); err != nil {
		return ""
	}
	return trigger.Name
}

func (c *RunTrigger) Setup(ctx core.SetupContext) error {
	config, err := decodeRunTriggerConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.TriggerID == "" {
		return fmt.Errorf("trigger is required")
	}

	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this component to run triggers")
	}

	if err := scheduleCloudBuildSetupIfNeeded(ctx.Integration); err != nil {
		return err
	}

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{"type": SubscriptionType})
	if err != nil {
		return fmt.Errorf("failed to subscribe to Cloud Build notifications: %w", err)
	}

	triggerName := fetchTriggerName(ctx, config)
	return ctx.Metadata.Set(RunTriggerNodeMetadata{
		SubscriptionID: subscriptionID.String(),
		TriggerName:    triggerName,
	})
}

func (c *RunTrigger) Execute(ctx core.ExecutionContext) error {
	config, err := decodeRunTriggerConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := config.ProjectID
	if projectID == "" {
		projectID = client.ProjectID()
	}
	if projectID == "" {
		return ctx.ExecutionState.Fail("error", "projectId is required")
	}

	body := buildRunTriggerBody(config)
	url := buildRunTriggerURL(projectID, config.TriggerID)
	responseBody, err := client.PostURL(context.Background(), url, body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to run trigger: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse response: %v", err))
	}

	buildData := extractBuildFromOperation(result)
	buildID := strings.TrimSpace(readBuildString(buildData, "id"))
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(buildData, "name"))
	}
	if buildID == "" {
		return ctx.ExecutionState.Fail("error", "Cloud Build run trigger response did not include a build ID")
	}

	if err := storeCreateBuildMetadata(ctx.Metadata, buildData, projectID); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to store build metadata: %v", err))
	}

	if err := ctx.ExecutionState.SetKV(createBuildExecutionKV, buildID); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to track build execution: %v", err))
	}

	if isTerminalBuildStatus(readBuildString(buildData, "status")) {
		return completeCreateBuildExecution(ctx.ExecutionState, buildData)
	}

	return ctx.Requests.ScheduleActionCall(createBuildPollAction, map[string]any{}, createBuildPollInterval)
}

func buildRunTriggerBody(config RunTriggerConfiguration) map[string]any {
	body := map[string]any{}
	if config.Ref == "" {
		return body
	}
	if name, ok := strings.CutPrefix(config.Ref, "refs/tags/"); ok {
		body["tagName"] = name
	} else if isCommitSHA(config.Ref) {
		body["commitSha"] = config.Ref
	} else {
		branch, _ := strings.CutPrefix(config.Ref, "refs/heads/")
		body["branchName"] = branch
	}
	return body
}

func isCommitSHA(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, c := range ref {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func (c *RunTrigger) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	config, err := decodeRunTriggerConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	var metadata CreateBuildExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	buildID := readBuildString(metadata.Build, "id")
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(metadata.Build, "name"))
	}
	if buildID == "" {
		return fmt.Errorf("build metadata is missing id")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	projectID := config.ProjectID
	if projectID == "" {
		projectID = client.ProjectID()
	}

	url := buildGetURL(projectID, buildID, readBuildString(metadata.Build, "name"))
	responseBody, err := client.GetURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to get build: %w", err)
	}

	var build map[string]any
	if err := json.Unmarshal(responseBody, &build); err != nil {
		return fmt.Errorf("failed to parse build response: %w", err)
	}

	if err := storeCreateBuildMetadata(ctx.Metadata, build, projectID); err != nil {
		return fmt.Errorf("failed to store build metadata: %w", err)
	}

	if !isTerminalBuildStatus(readBuildString(build, "status")) {
		return ctx.Requests.ScheduleActionCall(createBuildPollAction, map[string]any{}, createBuildPollInterval)
	}

	return completeCreateBuildExecution(ctx.ExecutionState, build)
}

func (c *RunTrigger) Actions() []core.Action {
	return []core.Action{
		{Name: createBuildPollAction, UserAccessible: false},
	}
}

func (c *RunTrigger) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case createBuildPollAction:
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *RunTrigger) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	if ctx.FindExecutionByKV == nil {
		return nil
	}

	build, ok := ctx.Message.(map[string]any)
	if !ok {
		return nil
	}

	buildID := readBuildString(build, "id")
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(build, "name"))
	}
	if buildID == "" {
		return nil
	}

	executionCtx, err := ctx.FindExecutionByKV(createBuildExecutionKV, buildID)
	if err != nil || executionCtx == nil {
		return err
	}

	if executionCtx.ExecutionState.IsFinished() {
		return nil
	}

	if err := storeCreateBuildMetadata(executionCtx.Metadata, build, readBuildString(build, "projectId")); err != nil {
		return fmt.Errorf("failed to store build metadata: %w", err)
	}

	if !isTerminalBuildStatus(readBuildString(build, "status")) {
		return nil
	}

	return completeCreateBuildExecution(executionCtx.ExecutionState, build)
}

func (c *RunTrigger) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RunTrigger) Cancel(ctx core.ExecutionContext) error {
	var metadata CreateBuildExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("decode run trigger metadata: %w", err)
	}

	buildID := readBuildString(metadata.Build, "id")
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(metadata.Build, "name"))
	}
	if buildID == "" {
		return nil
	}

	if isTerminalBuildStatus(readBuildString(metadata.Build, "status")) {
		return nil
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create GCP client: %w", err)
	}

	projectID := readBuildString(metadata.Build, "projectId")
	if projectID == "" {
		projectID = client.ProjectID()
	}

	cancelURL := buildCancelURL(projectID, buildID, readBuildString(metadata.Build, "name"))
	if _, err := client.PostURL(context.Background(), cancelURL, map[string]any{}); err != nil {
		return fmt.Errorf("cancel Cloud Build build %s: %w", buildID, err)
	}

	cancelledBuild := copyBuildMetadata(metadata.Build)
	cancelledBuild["status"] = "CANCELLED"
	if err := storeCreateBuildMetadata(ctx.Metadata, cancelledBuild, projectID); err != nil {
		return fmt.Errorf("store cancelled build metadata: %w", err)
	}

	return nil
}

func (c *RunTrigger) Cleanup(_ core.SetupContext) error { return nil }
func (c *RunTrigger) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
