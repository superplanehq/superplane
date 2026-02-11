package codebuild

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	BuildPayloadType  = "aws.codebuild.build"
	BuildPollInterval = 10 * time.Second
)

type RunBuild struct{}

type RunBuildConfiguration struct {
	Region        string `json:"region" mapstructure:"region"`
	Project       string `json:"project" mapstructure:"project"`
	SourceVersion string `json:"sourceVersion" mapstructure:"sourceVersion"`
}

type RunBuildMetadata struct {
	Region      string `json:"region" mapstructure:"region"`
	ProjectName string `json:"projectName" mapstructure:"projectName"`
	BuildID     string `json:"buildId" mapstructure:"buildId"`
}

func (c *RunBuild) Name() string {
	return "aws.codebuild.runBuild"
}

func (c *RunBuild) Label() string {
	return "CodeBuild • Run Build"
}

func (c *RunBuild) Description() string {
	return "Start an AWS CodeBuild build and wait for it to finish"
}

func (c *RunBuild) Documentation() string {
	return `The Run Build component starts a build in AWS CodeBuild and waits until the build reaches a terminal state.

## Use Cases

- **CI orchestration**: Trigger builds from workflow automation
- **Release pipelines**: Block downstream steps until build completion
- **Quality gates**: Fail workflow execution automatically when builds fail

## How It Works

1. Starts a new build for the selected CodeBuild project
2. Polls CodeBuild for the build status until completion
3. Emits build details when the build succeeds
4. Fails the node if the build finishes with a failed status

## Configuration

- **Region**: AWS region where the CodeBuild project exists
- **Project**: CodeBuild project name
- **Source Version** (optional): Specific source revision, branch, or commit to build`
}

func (c *RunBuild) Icon() string {
	return "aws"
}

func (c *RunBuild) Color() string {
	return "gray"
}

func (c *RunBuild) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunBuild) Configuration() []configuration.Field {
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
			Description: "CodeBuild project name",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
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
		},
		{
			Name:        "sourceVersion",
			Label:       "Source Version",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional source revision, branch, or commit to build",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "project",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *RunBuild) Setup(ctx core.SetupContext) error {
	_, err := decodeRunBuildConfiguration(ctx.Configuration)
	return err
}

func (c *RunBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunBuild) Execute(ctx core.ExecutionContext) error {
	config, err := decodeRunBuildConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	projectName, err := projectNameFromRef(config.Project)
	if err != nil {
		return err
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	build, err := client.StartBuild(projectName, config.SourceVersion)
	if err != nil {
		return fmt.Errorf("failed to start build: %w", err)
	}

	if strings.TrimSpace(build.ID) == "" {
		return fmt.Errorf("build ID is missing")
	}

	if finished, succeeded := isBuildTerminalStatus(build.BuildStatus); finished {
		if !succeeded {
			return buildFailureError(build)
		}
		return c.emitBuild(ctx.ExecutionState, build)
	}

	if err := ctx.Metadata.Set(RunBuildMetadata{
		Region:      config.Region,
		ProjectName: projectName,
		BuildID:     build.ID,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("pollBuild", map[string]any{}, BuildPollInterval)
}

func (c *RunBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "pollBuild",
			Description: "Poll build status",
		},
	}
}

func (c *RunBuild) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "pollBuild":
		return c.pollBuild(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *RunBuild) pollBuild(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunBuildMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if strings.TrimSpace(metadata.BuildID) == "" {
		return fmt.Errorf("build ID is missing from metadata")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, metadata.Region)
	builds, err := client.BatchGetBuilds([]string{metadata.BuildID})
	if err != nil {
		return fmt.Errorf("failed to get build status: %w", err)
	}

	if len(builds) == 0 {
		return fmt.Errorf("build not found: %s", metadata.BuildID)
	}

	build := &builds[0]
	if finished, succeeded := isBuildTerminalStatus(build.BuildStatus); finished {
		if !succeeded {
			return buildFailureError(build)
		}
		return c.emitBuild(ctx.ExecutionState, build)
	}

	return ctx.Requests.ScheduleActionCall("pollBuild", map[string]any{}, BuildPollInterval)
}

func (c *RunBuild) emitBuild(executionState core.ExecutionStateContext, build *Build) error {
	return executionState.Emit(core.DefaultOutputChannel.Name, BuildPayloadType, []any{build})
}

func (c *RunBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RunBuild) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeRunBuildConfiguration(configuration any) (RunBuildConfiguration, error) {
	config := RunBuildConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return RunBuildConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Project = strings.TrimSpace(config.Project)
	config.SourceVersion = strings.TrimSpace(config.SourceVersion)

	if config.Region == "" {
		return RunBuildConfiguration{}, fmt.Errorf("region is required")
	}

	if config.Project == "" {
		return RunBuildConfiguration{}, fmt.Errorf("project is required")
	}

	if _, err := projectNameFromRef(config.Project); err != nil {
		return RunBuildConfiguration{}, err
	}

	return config, nil
}

func isBuildTerminalStatus(status string) (finished bool, succeeded bool) {
	normalized := strings.ToUpper(strings.TrimSpace(status))

	switch normalized {
	case "SUCCEEDED":
		return true, true

	case "FAILED", "FAULT", "TIMED_OUT", "STOPPED":
		return true, false

	default:
		return false, false
	}
}

func buildFailureError(build *Build) error {
	if build == nil {
		return fmt.Errorf("CodeBuild build finished with status UNKNOWN")
	}

	status := strings.ToUpper(strings.TrimSpace(build.BuildStatus))
	if status == "" {
		status = "UNKNOWN"
	}

	if build != nil && build.Logs != nil && strings.TrimSpace(build.Logs.DeepLink) != "" {
		return fmt.Errorf(
			"CodeBuild build %s finished with status %s: %s",
			strings.TrimSpace(build.ID),
			status,
			strings.TrimSpace(build.Logs.DeepLink),
		)
	}

	return fmt.Errorf("CodeBuild build %s finished with status %s", strings.TrimSpace(build.ID), status)
}
