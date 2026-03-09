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

type StopBuild struct{}

type StopBuildSpec struct {
	Region  string `json:"region" mapstructure:"region"`
	BuildID string `json:"buildId" mapstructure:"buildId"`
}

func (c *StopBuild) Name() string {
	return "aws.codebuild.stopBuild"
}

func (c *StopBuild) Label() string {
	return "CodeBuild • Stop Build"
}

func (c *StopBuild) Description() string {
	return "Stop a running AWS CodeBuild build"
}

func (c *StopBuild) Documentation() string {
	return `The Stop Build component stops a running AWS CodeBuild build.

## Use Cases

- **Build cancellation**: Stop builds that are no longer needed
- **Resource management**: Cancel running builds to free up build resources
- **Workflow control**: Stop builds as part of error handling or workflow branching

## Configuration

- **Region**: AWS region where the build is running
- **Build ID**: The full build ID to stop (e.g., "project-name:build-uuid")

## Output

Emits the stopped build details including project name, build ID, and final status.`
}

func (c *StopBuild) Icon() string {
	return "aws"
}

func (c *StopBuild) Color() string {
	return "orange"
}

func (c *StopBuild) Configuration() []configuration.Field {
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
			Description: "The full CodeBuild build ID to stop (e.g., project-name:build-uuid)",
		},
	}
}

func (c *StopBuild) Setup(ctx core.SetupContext) error {
	spec := StopBuildSpec{}
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

func (c *StopBuild) Execute(ctx core.ExecutionContext) error {
	spec := StopBuildSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, spec.Region)

	build, err := client.StopBuild(spec.BuildID)
	if err != nil {
		return fmt.Errorf("failed to stop build: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codebuild.build",
		[]any{build},
	)
}

func (c *StopBuild) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *StopBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StopBuild) Actions() []core.Action {
	return []core.Action{}
}

func (c *StopBuild) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *StopBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *StopBuild) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *StopBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}
