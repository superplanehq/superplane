package codebuild

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type GetBuildStatus struct{}

type GetBuildStatusSpec struct {
	Region  string `json:"region" mapstructure:"region"`
	BuildID string `json:"buildId" mapstructure:"buildId"`
}

func (c *GetBuildStatus) Name() string {
	return "aws.codebuild.getBuildStatus"
}

func (c *GetBuildStatus) Label() string {
	return "CodeBuild • Get Build Status"
}

func (c *GetBuildStatus) Description() string {
	return "Retrieve the status and details of an AWS CodeBuild build"
}

func (c *GetBuildStatus) Documentation() string {
	return `The Get Build Status component retrieves the full details of an AWS CodeBuild build.

## Use Cases

- **Build inspection**: Fetch build status, phases, and configuration
- **Workflow branching**: Route workflow based on build status
- **Monitoring**: Check build progress and results

## Configuration

- **Region**: AWS region where the build exists
- **Build ID**: The full build ID to retrieve (e.g., "project-name:build-uuid")

## Output

Emits the full build details including:
- Build ID, ARN, and build number
- Build status and current phase
- Project name and source configuration
- Build start and end times
- Build logs location`
}

func (c *GetBuildStatus) Icon() string {
	return "aws"
}

func (c *GetBuildStatus) Color() string {
	return "orange"
}

func (c *GetBuildStatus) Configuration() []configuration.Field {
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
			Name:        "buildId",
			Label:       "Build ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The full CodeBuild build ID to retrieve (e.g., project-name:build-uuid)",
		},
	}
}

func (c *GetBuildStatus) Setup(ctx core.SetupContext) error {
	spec := GetBuildStatusSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Region) == "" {
		return fmt.Errorf("region is required")
	}

	if strings.TrimSpace(spec.BuildID) == "" {
		return fmt.Errorf("build ID is required")
	}

	return nil
}

func (c *GetBuildStatus) Execute(ctx core.ExecutionContext) error {
	spec := GetBuildStatusSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	builds, err := client.BatchGetBuilds([]string{spec.BuildID})
	if err != nil {
		return fmt.Errorf("failed to get build: %w", err)
	}

	if len(builds) == 0 {
		return fmt.Errorf("build not found: %s", spec.BuildID)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codebuild.build",
		[]any{builds[0]},
	)
}

func (c *GetBuildStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetBuildStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetBuildStatus) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetBuildStatus) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetBuildStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetBuildStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetBuildStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}
