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
	ProjectID  string `json:"projectId" mapstructure:"projectId"`
	TriggerID  string `json:"trigger" mapstructure:"trigger"`
	BranchName string `json:"branchName" mapstructure:"branchName"`
	TagName    string `json:"tagName" mapstructure:"tagName"`
	CommitSHA  string `json:"commitSha" mapstructure:"commitSha"`
}

type RunTriggerNodeMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId,omitempty"`
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
- **Branch Name**: Override the branch to build from. Mutually exclusive with Tag Name and Commit SHA. Leave empty to use the trigger's default.
- **Tag Name**: Override the tag to build from. Mutually exclusive with Branch Name and Commit SHA.
- **Commit SHA**: Override the commit SHA to build from. Mutually exclusive with Branch Name and Tag Name.
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
			Name:        "projectId",
			Label:       "Project ID Override",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the GCP project ID from the integration. Leave empty to use the integration's project.",
		},
		{
			Name:        "trigger",
			Label:       "Trigger",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Cloud Build trigger to run.",
			Placeholder: "Select a trigger",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTrigger,
					Parameters: []configuration.ParameterRef{
						{Name: "projectId", ValueFrom: &configuration.ParameterValueFrom{Field: "projectId"}},
					},
				},
			},
		},
		{
			Name:        "branchName",
			Label:       "Branch Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the branch to build from. Mutually exclusive with Tag Name and Commit SHA. Leave empty to use the trigger's default.",
			Placeholder: "e.g. main",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the tag to build from. Mutually exclusive with Branch Name and Commit SHA.",
			Placeholder: "e.g. v1.0.0",
		},
		{
			Name:        "commitSha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the commit SHA to build from. Mutually exclusive with Branch Name and Tag Name.",
			Placeholder: "e.g. 5d7363a99d19e45830e1bc9622d2e4fa72d7229f",
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
	config.BranchName = strings.TrimSpace(config.BranchName)
	config.TagName = strings.TrimSpace(config.TagName)
	config.CommitSHA = strings.TrimSpace(config.CommitSHA)
	return config, nil
}

func (c *RunTrigger) Setup(ctx core.SetupContext) error {
	config, err := decodeRunTriggerConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.TriggerID == "" {
		return fmt.Errorf("trigger is required")
	}

	set := 0
	if config.BranchName != "" {
		set++
	}
	if config.TagName != "" {
		set++
	}
	if config.CommitSHA != "" {
		set++
	}
	if set > 1 {
		return fmt.Errorf("branchName, tagName, and commitSha are mutually exclusive")
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

	return ctx.Metadata.Set(RunTriggerNodeMetadata{
		SubscriptionID: subscriptionID.String(),
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
	if config.BranchName != "" {
		body["branchName"] = config.BranchName
	} else if config.TagName != "" {
		body["tagName"] = config.TagName
	} else if config.CommitSHA != "" {
		body["commitSha"] = config.CommitSHA
	}
	return body
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

func (c *RunTrigger) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
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
