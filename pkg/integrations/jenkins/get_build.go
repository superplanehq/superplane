package jenkins

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PayloadTypeGetBuild = "jenkins.build"

type GetBuild struct{}

type GetBuildSpec struct {
	JobName     string `json:"jobName" mapstructure:"jobName"`
	BuildNumber int    `json:"buildNumber" mapstructure:"buildNumber"`
}

func (g *GetBuild) Name() string {
	return "jenkins.getBuild"
}

func (g *GetBuild) Label() string {
	return "Get Build Status"
}

func (g *GetBuild) Description() string {
	return "Get the status of a Jenkins build"
}

func (g *GetBuild) Documentation() string {
	return `The Get Build Status component reads a Jenkins build's status back into the workflow.

## Use Cases

- **Build monitoring**: Check whether a build is still running or has finished
- **Conditional logic**: Branch a workflow based on a build's result
- **Reporting**: Pull build metadata (URL, duration) into downstream steps

## Configuration

- **Job Name**: The name of the Jenkins job (top-level jobs only in this version)
- **Build Number**: The build number to look up (supports expressions)

## Output

- **building**: Whether the build is still running
- **result**: The build result (` + "`SUCCESS`" + `, ` + "`FAILURE`" + `, etc.), or ` + "`null`" + ` while running
- **number**: The build number
- **url**: The Jenkins build URL
- **durationMs**: The build duration in milliseconds (0 while running)
`
}

func (g *GetBuild) Icon() string {
	return "jenkins"
}

func (g *GetBuild) Color() string {
	return "gray"
}

func (g *GetBuild) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "jobName",
			Label:       "Job Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the Jenkins job",
		},
		{
			Name:        "buildNumber",
			Label:       "Build Number",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "The build number to look up",
		},
	}
}

func decodeGetBuildConfiguration(config any) (GetBuildSpec, error) {
	spec := GetBuildSpec{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return GetBuildSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.JobName = strings.TrimSpace(spec.JobName)
	if spec.JobName == "" {
		return GetBuildSpec{}, fmt.Errorf("jobName is required")
	}

	if spec.BuildNumber <= 0 {
		return GetBuildSpec{}, fmt.Errorf("a valid buildNumber is required")
	}

	return spec, nil
}

func (g *GetBuild) Setup(ctx core.SetupContext) error {
	_, err := decodeGetBuildConfiguration(ctx.Configuration)
	return err
}

func (g *GetBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetBuild) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetBuildConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	build, err := client.GetBuild(spec.JobName, spec.BuildNumber)
	if err != nil {
		return fmt.Errorf("failed to get build: %w", err)
	}

	payload := map[string]any{
		"building":   build.Building,
		"result":     build.Result,
		"number":     build.Number,
		"url":        build.URL,
		"durationMs": build.Duration,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeGetBuild, []any{payload})
}

func (g *GetBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetBuild) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetBuild) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetBuild) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
