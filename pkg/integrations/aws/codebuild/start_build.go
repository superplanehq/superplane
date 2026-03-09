package codebuild

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	BuildPayloadType = "aws.codebuild.build.finished"

	PassedOutputChannel = "passed"
	FailedOutputChannel = "failed"

	BuildStatusInProgress = "IN_PROGRESS"
	BuildStatusSucceeded  = "SUCCEEDED"
	BuildStatusFailed     = "FAILED"
	BuildStatusStopped    = "STOPPED"
	BuildStatusTimedOut   = "TIMED_OUT"
	BuildStatusFault      = "FAULT"

	PollInterval = 5 * time.Minute
)

type StartBuild struct{}

type StartBuildSpec struct {
	Region               string                `json:"region" mapstructure:"region"`
	Project              string                `json:"project" mapstructure:"project"`
	EnvironmentVariables []EnvironmentVariable `json:"environmentVariables" mapstructure:"environmentVariables"`
}

type StartBuildNodeMetadata struct {
	Region         string           `json:"region,omitempty" mapstructure:"region,omitempty"`
	Project        *ProjectMetadata `json:"project" mapstructure:"project"`
	SubscriptionID string           `json:"subscriptionId,omitempty" mapstructure:"subscriptionId,omitempty"`
}

type ProjectMetadata struct {
	Name string `json:"name"`
}

type StartBuildExecutionMetadata struct {
	Project *ProjectMetadata `json:"project" mapstructure:"project"`
	Build   *BuildMetadata   `json:"build" mapstructure:"build"`
	Extra   map[string]any   `json:"extra,omitempty" mapstructure:"extra,omitempty"`
}

type BuildMetadata struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func terminalBuildStatus(status string) bool {
	switch status {
	case BuildStatusSucceeded, BuildStatusFailed, BuildStatusStopped, BuildStatusTimedOut, BuildStatusFault:
		return true
	default:
		return false
	}
}

func buildOutputPayload(projectName, buildID, status string, detail map[string]any) map[string]any {
	return map[string]any{
		"build": map[string]any{
			"project": projectName,
			"id":      buildID,
			"status":  status,
		},
		"detail": detail,
	}
}

func (s *StartBuild) Name() string {
	return "aws.codebuild.startBuild"
}

func (s *StartBuild) Label() string {
	return "CodeBuild • Start Build"
}

func (s *StartBuild) Description() string {
	return "Start an AWS CodeBuild build and wait for it to complete"
}

func (s *StartBuild) Documentation() string {
	return `The Start Build component triggers an AWS CodeBuild build and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger builds from SuperPlane workflows
- **Build automation**: Run CodeBuild projects as part of workflow automation
- **Multi-stage deployments**: Coordinate complex build and deploy pipelines
- **Workflow chaining**: Chain multiple CodeBuild projects together

## How It Works

1. Starts a CodeBuild build with the specified project name
2. Waits for the build to complete (monitored via EventBridge webhook and polling)
3. Routes execution based on build result:
   - **Passed channel**: Build completed successfully
   - **Failed channel**: Build failed, timed out, or was stopped

## Configuration

- **Region**: AWS region where the project exists
- **Project**: CodeBuild project name to build
- **Environment Variables**: Optional environment variable overrides for the build

## Output Channels

- **Passed**: Emitted when build completes successfully
- **Failed**: Emitted when build fails, times out, or is stopped

## Notes

- The component automatically sets up EventBridge monitoring for build completion
- Falls back to polling if webhook doesn't arrive
- Can be cancelled, which will stop the running build`
}

func (s *StartBuild) Icon() string {
	return "aws"
}

func (s *StartBuild) Color() string {
	return "orange"
}

func (s *StartBuild) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  PassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  FailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (s *StartBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "CodeBuild project to build",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "codebuild.project",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:      "environmentVariables",
			Label:     "Environment Variables",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "name",
								Label:              "Name",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func (s *StartBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (s *StartBuild) Setup(ctx core.SetupContext) error {
	spec := StartBuildSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Region == "" {
		return fmt.Errorf("region is required")
	}
	if spec.Project == "" {
		return fmt.Errorf("project is required")
	}

	metadata := StartBuildNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		metadata = StartBuildNodeMetadata{}
	}

	if metadata.SubscriptionID != "" && metadata.Project != nil && spec.Project == metadata.Project.Name && spec.Region == metadata.Region {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var foundProject *ProjectMetadata
	for _, p := range projects {
		if p == spec.Project {
			foundProject = &ProjectMetadata{
				Name: p,
			}
			break
		}
	}

	if foundProject == nil {
		return fmt.Errorf("project not found: %s", spec.Project)
	}

	source := "aws.codebuild"
	detailType := "CodeBuild Build State Change"

	hasRule, err := common.HasEventBridgeRule(ctx.Logger, ctx.Integration, source, spec.Region, detailType)
	if err != nil {
		ctx.Logger.Warnf("Failed to check EventBridge rule availability: %v", err)
	}

	if !hasRule {
		err = ctx.Integration.ScheduleActionCall(
			"provisionRule",
			common.ProvisionRuleParameters{
				Region:     spec.Region,
				Source:     source,
				DetailType: detailType,
			},
			time.Second,
		)
		if err != nil {
			ctx.Logger.Warnf("Failed to schedule EventBridge rule provisioning: %v", err)
		}
	}

	subscriptionID, err := ctx.Integration.Subscribe(&common.EventBridgeEvent{
		Region:     spec.Region,
		DetailType: "CodeBuild Build State Change",
		Source:     "aws.codebuild",
	})

	nodeMetadata := StartBuildNodeMetadata{
		Region:  spec.Region,
		Project: foundProject,
	}

	if err != nil {
		ctx.Logger.Warnf("Failed to subscribe to CodeBuild events: %v", err)
	} else {
		nodeMetadata.SubscriptionID = subscriptionID.String()
	}

	err = ctx.Metadata.Set(nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

func (s *StartBuild) Execute(ctx core.ExecutionContext) error {
	spec := StartBuildSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	nodeMetadata := StartBuildNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if nodeMetadata.Project == nil {
		return fmt.Errorf("project metadata not found - component may not be properly set up")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	build, err := client.StartBuild(nodeMetadata.Project.Name, spec.EnvironmentVariables)
	if err != nil {
		return fmt.Errorf("failed to start build: %w", err)
	}

	ctx.Logger.Infof("Started build - project=%s, build=%s", nodeMetadata.Project.Name, build.ID)

	err = ctx.Metadata.Set(StartBuildExecutionMetadata{
		Project: nodeMetadata.Project,
		Build: &BuildMetadata{
			ID:     build.ID,
			Status: BuildStatusInProgress,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	err = ctx.ExecutionState.SetKV("build_id", build.ID)
	if err != nil {
		return fmt.Errorf("failed to set build ID: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (s *StartBuild) Cancel(ctx core.ExecutionContext) error {
	metadata := StartBuildExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Build == nil || metadata.Build.ID == "" {
		return nil
	}

	if metadata.Project == nil {
		return nil
	}

	if terminalBuildStatus(metadata.Build.Status) {
		return nil
	}

	spec := StartBuildSpec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("Failed to get AWS credentials for cancellation: %v", err)
		return nil
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)
	_, err = client.StopBuild(metadata.Build.ID)
	if err != nil {
		ctx.Logger.Warnf("Failed to stop build: %v", err)
		return nil
	}

	ctx.Logger.Infof("Stopped build %s", metadata.Build.ID)
	return nil
}

func (s *StartBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *StartBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
			Description:    "Check build status",
		},
		{
			Name:           "finish",
			UserAccessible: true,
			Description:    "Manually finish the execution",
			Parameters: []configuration.Field{
				{
					Name:     "data",
					Type:     configuration.FieldTypeObject,
					Required: false,
					Default:  map[string]any{},
				},
			},
		},
	}
}

func (s *StartBuild) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return s.poll(ctx)
	case "finish":
		return s.finish(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (s *StartBuild) poll(ctx core.ActionContext) error {
	spec := StartBuildSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := StartBuildExecutionMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Project == nil {
		return fmt.Errorf("project metadata not found - component may not be properly set up")
	}

	if metadata.Build == nil {
		return fmt.Errorf("build metadata not found - component may not have started properly")
	}

	if terminalBuildStatus(metadata.Build.Status) {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	builds, err := client.BatchGetBuilds([]string{metadata.Build.ID})
	if err != nil {
		return fmt.Errorf("failed to get build status: %w", err)
	}

	if len(builds) == 0 {
		return fmt.Errorf("build not found: %s", metadata.Build.ID)
	}

	build := builds[0]

	if build.BuildStatus == BuildStatusInProgress {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	metadata.Build.Status = build.BuildStatus
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	detail := map[string]any{
		"project-name":  metadata.Project.Name,
		"build-id":      build.ID,
		"build-status":  build.BuildStatus,
		"current-phase": build.CurrentPhase,
	}

	outputPayload := buildOutputPayload(metadata.Project.Name, build.ID, build.BuildStatus, detail)

	if build.BuildStatus == BuildStatusSucceeded {
		return ctx.ExecutionState.Emit(PassedOutputChannel, BuildPayloadType, []any{outputPayload})
	}

	return ctx.ExecutionState.Emit(FailedOutputChannel, BuildPayloadType, []any{outputPayload})
}

func (s *StartBuild) finish(ctx core.ActionContext) error {
	metadata := StartBuildExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	if metadata.Build != nil && terminalBuildStatus(metadata.Build.Status) {
		return fmt.Errorf("build already finished")
	}

	data, ok := ctx.Parameters["data"]
	if !ok {
		data = map[string]any{}
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("data parameter is invalid")
	}

	if metadata.Project == nil {
		return fmt.Errorf("project metadata not found - component may not be properly set up")
	}

	metadata.Extra = dataMap
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	outputPayload := map[string]any{
		"build": map[string]any{
			"project": metadata.Project.Name,
		},
		"manual": true,
		"data":   dataMap,
	}

	if metadata.Build != nil {
		outputPayload["build"].(map[string]any)["id"] = metadata.Build.ID
		outputPayload["build"].(map[string]any)["status"] = metadata.Build.Status
	}

	return ctx.ExecutionState.Emit(PassedOutputChannel, BuildPayloadType, []any{outputPayload})
}

func (s *StartBuild) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	event := common.EventBridgeEvent{}
	err := mapstructure.Decode(ctx.Message, &event)
	if err != nil {
		return fmt.Errorf("failed to decode EventBridge event: %w", err)
	}

	projectName, ok := event.Detail["project-name"]
	if !ok {
		return fmt.Errorf("missing project-name in event detail")
	}

	name, ok := projectName.(string)
	if !ok {
		return fmt.Errorf("invalid project-name in event detail")
	}

	metadata := StartBuildNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if metadata.Project == nil {
		return nil
	}

	if name != metadata.Project.Name {
		ctx.Logger.Infof("Skipping event for project %s, expected %s", name, metadata.Project.Name)
		return nil
	}

	status, ok := event.Detail["build-status"].(string)
	if !ok {
		return nil
	}

	if !terminalBuildStatus(status) {
		return nil
	}

	buildID, ok := event.Detail["build-id"].(string)
	if !ok || buildID == "" {
		return fmt.Errorf("missing build-id in EventBridge event detail")
	}

	// Extract the short build ID (after the colon) to match the full build ID
	// CodeBuild EventBridge events use "project:build-uuid" format for build-id
	if ctx.FindExecutionByKV == nil {
		ctx.Logger.Warnf("FindExecutionByKV not available, falling back to event emission")
		return ctx.Events.Emit(BuildPayloadType, ctx.Message)
	}

	executionCtx, err := ctx.FindExecutionByKV("build_id", buildID)
	if err != nil {
		ctx.Logger.Warnf("Failed to find execution for build_id=%s: %v", buildID, err)
		return nil
	}

	if executionCtx == nil {
		ctx.Logger.Infof("No execution found for build_id=%s, ignoring", buildID)
		return nil
	}

	execMetadata := StartBuildExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &execMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode execution metadata: %w", err)
	}

	if execMetadata.Build == nil {
		return nil
	}

	if terminalBuildStatus(execMetadata.Build.Status) {
		return nil
	}

	if executionCtx.ExecutionState.IsFinished() {
		return nil
	}

	execMetadata.Build.Status = status
	err = executionCtx.Metadata.Set(execMetadata)
	if err != nil {
		return fmt.Errorf("failed to update execution metadata: %w", err)
	}

	outputPayload := buildOutputPayload(metadata.Project.Name, buildID, status, event.Detail)

	if status == BuildStatusSucceeded {
		return executionCtx.ExecutionState.Emit(PassedOutputChannel, BuildPayloadType, []any{outputPayload})
	}

	return executionCtx.ExecutionState.Emit(FailedOutputChannel, BuildPayloadType, []any{outputPayload})
}

func (s *StartBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}
