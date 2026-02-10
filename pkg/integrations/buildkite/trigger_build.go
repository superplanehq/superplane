package buildkite

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	PassedOutputChannel = "passed"
	FailedOutputChannel = "failed"
	PayloadType         = "buildkite.build.finished"
	PollInterval        = 30 * time.Second

	BuildStatePassed   = "passed"
	BuildStateFailed   = "failed"
	BuildStateBlocked  = "blocked"
	BuildStateCanceled = "canceled"
	BuildStateSkipped  = "skipped"
	BuildStateNotRun   = "not_run"
)

var TerminalStates = map[string]bool{
	BuildStatePassed:   true,
	BuildStateFailed:   true,
	BuildStateBlocked:  true,
	BuildStateCanceled: true,
	BuildStateSkipped:  true,
	BuildStateNotRun:   true,
}

type TriggerBuild struct{}

type TriggerBuildNodeMetadata struct {
	Organization string `json:"organization"`
	Pipeline     string `json:"pipeline"`
}

type TriggerBuildExecutionMetadata struct {
	BuildID      string         `json:"build_id" mapstructure:"build_id"`
	BuildNumber  int            `json:"build_number" mapstructure:"build_number"`
	WebURL       string         `json:"web_url" mapstructure:"web_url"`
	Organization string         `json:"organization" mapstructure:"organization"`
	Pipeline     string         `json:"pipeline" mapstructure:"pipeline"`
	State        string         `json:"state" mapstructure:"state"`
	Blocked      bool           `json:"blocked" mapstructure:"blocked"`
	Extra        map[string]any `json:"extra,omitempty" mapstructure:"extra,omitempty"`
}

type TriggerBuildSpec struct {
	Organization string            `json:"organization"`
	Pipeline     string            `json:"pipeline"`
	Branch       string            `json:"branch"`
	Commit       string            `json:"commit"`
	Message      string            `json:"message,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Metadata     map[string]string `json:"meta_data,omitempty"`
}

func (r *TriggerBuild) Name() string {
	return "buildkite.triggerBuild"
}

func (r *TriggerBuild) Label() string {
	return "Trigger Build"
}

func (r *TriggerBuild) Description() string {
	return `Trigger a Buildkite build and wait for completion using polling mechanism

This component periodically polls the Buildkite API to check build status instead of relying on webhooks.

## How It Works

1. **Creates Build**: Triggers a build via Buildkite API
2. **Sets Correlation**: Stores build ID for tracking
3. **Starts Polling**: Schedules periodic status checks
4. **Detects Completion**: Emits to success/failure channels when build finishes

## Configuration

- **Organization**: Select Buildkite organization
- **Pipeline**: Select pipeline to trigger
- **Branch**: Git branch to build
- **Commit**: Git commit SHA (optional, defaults to HEAD)
- **Message**: Optional build message
- **Environment Variables**: Optional env vars for the build
- **Metadata**: Optional metadata for the build

## Output Channels

- **Passed**: Build completed successfully
- **Failed**: Build failed, was cancelled, or was blocked
`
}

func (r *TriggerBuild) Icon() string {
	return "workflow"
}

func (r *TriggerBuild) Color() string {
	return "gray"
}

func (r *TriggerBuild) OutputChannels(configuration any) []core.OutputChannel {
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

func (r *TriggerBuild) ExampleOutput() map[string]any {
	return map[string]any{
		"build": map[string]any{
			"id":      "12345678-1234-1234-123456789012",
			"number":  123,
			"state":   "passed",
			"web_url": "https://buildkite.com/example-org/example-pipeline/builds/123",
			"commit":  "a1b2c3d4e5f678901234567890abcd",
			"branch":  "main",
			"message": "Triggered by SuperPlane",
			"blocked": false,
		},
		"pipeline": map[string]any{
			"id": "example-pipeline",
		},
		"organization": map[string]any{
			"id": "example-org",
		},
	}
}

func (r *TriggerBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "organization",
			Label:    "Organization",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "organization",
				},
			},
		},
		{
			Name:     "pipeline",
			Label:    "Pipeline",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pipeline",
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
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Git branch to run build on",
			Placeholder: "e.g. main, develop",
		},
		{
			Name:        "commit",
			Label:       "Commit",
			Type:        configuration.FieldTypeString,
			Description: "Git commit SHA to build (optional, defaults to HEAD)",
			Placeholder: "e.g. a1b2c3d4e5f678901234567890abcd",
		},
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeString,
			Description: "Optional build message",
			Placeholder: "e.g. Triggered by SuperPlane workflow",
		},
		{
			Name:  "env",
			Label: "Environment Variables",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Environment Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		{
			Name:  "meta_data",
			Label: "Metadata",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Metadata Item",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
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

func (r *TriggerBuild) Setup(ctx core.SetupContext) error {
	config := TriggerBuildSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	if config.Organization == "" || config.Pipeline == "" {
		return fmt.Errorf("organization and pipeline are required")
	}

	metadata := TriggerBuildNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Organization == config.Organization && metadata.Pipeline == config.Pipeline {
		return nil
	}

	err = ctx.Metadata.Set(TriggerBuildNodeMetadata{
		Organization: config.Organization,
		Pipeline:     config.Pipeline,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *TriggerBuild) Execute(ctx core.ExecutionContext) error {
	spec := TriggerBuildSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	commit := spec.Commit
	if commit == "" {
		commit = "HEAD"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	buildReq := CreateBuildRequest{
		Commit:   commit,
		Branch:   spec.Branch,
		Message:  spec.Message,
		Env:      spec.Env,
		Metadata: spec.Metadata,
	}

	if buildReq.Metadata == nil {
		buildReq.Metadata = make(map[string]string)
	}
	buildReq.Metadata["superplane_execution_id"] = ctx.ID.String()
	buildReq.Metadata["superplane_workflow_id"] = ctx.WorkflowID

	build, err := client.CreateBuild(spec.Organization, spec.Pipeline, buildReq)
	if err != nil {
		return fmt.Errorf("error creating build: %v", err)
	}

	err = ctx.Metadata.Set(TriggerBuildExecutionMetadata{
		BuildID:      build.ID,
		BuildNumber:  build.Number,
		WebURL:       build.WebURL,
		Organization: spec.Organization,
		Pipeline:     spec.Pipeline,
		State:        build.State,
		Blocked:      build.Blocked,
	})
	if err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (r *TriggerBuild) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *TriggerBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
		{
			Name:           "finish",
			UserAccessible: true,
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

func (r *TriggerBuild) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return r.poll(ctx)
	case "finish":
		return r.finish(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (r *TriggerBuild) poll(ctx core.ActionContext) error {
	spec := TriggerBuildSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := TriggerBuildExecutionMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	targetBuild, err := client.GetBuild(metadata.Organization, metadata.Pipeline, metadata.BuildNumber)
	if err != nil {
		return err
	}

	if targetBuild.ID != metadata.BuildID {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	if TerminalStates[targetBuild.State] {
		payload := map[string]any{
			"build": map[string]any{
				"id":      targetBuild.ID,
				"number":  targetBuild.Number,
				"state":   targetBuild.State,
				"web_url": targetBuild.WebURL,
				"commit":  targetBuild.Commit,
				"branch":  targetBuild.Branch,
				"message": targetBuild.Message,
				"blocked": targetBuild.Blocked,
			},
			"pipeline": map[string]any{
				"id": metadata.Pipeline,
			},
			"organization": map[string]any{
				"id": metadata.Organization,
			},
		}

		metadata.State = targetBuild.State
		metadata.Blocked = targetBuild.Blocked
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}

		isSuccess := targetBuild.State == BuildStatePassed && !targetBuild.Blocked
		if isSuccess {
			return ctx.ExecutionState.Emit(PassedOutputChannel, PayloadType, []any{payload})
		}

		if err := ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload}); err != nil {
			return err
		}
		failMessage := fmt.Sprintf("build %s finished with state: %s", targetBuild.ID, targetBuild.State)
		if targetBuild.Blocked {
			failMessage = fmt.Sprintf("build %s finished with state: %s (blocked)", targetBuild.ID, targetBuild.State)
		}
		return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, failMessage)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (r *TriggerBuild) finish(ctx core.ActionContext) error {
	data, _ := ctx.Parameters["data"]
	dataMap, _ := data.(map[string]any)

	metadata := TriggerBuildExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return err
	}

	if dataMap != nil {
		metadata.Extra = dataMap
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
	}

	return nil
}

func (r *TriggerBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (r *TriggerBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *TriggerBuild) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var config struct{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

func (r *TriggerBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (r *TriggerBuild) Documentation() string {
	return `Trigger a Buildkite build and wait for completion using polling mechanism.

## How It Works

1. **Creates Build**: Triggers a build via Buildkite API
2. **Sets Correlation**: Stores build ID for tracking
3. **Starts Polling**: Schedules periodic status checks
4. **Detects Completion**: Emits to success/failure channels when build finishes

## Why Polling Instead of Webhooks?

This component uses polling instead of webhooks to work around the architectural limitation where Buildkite integration uses a shared webhook for all events. The polling mechanism provides reliable completion detection even when webhook routing is not available.

## Configuration

- **Organization**: Select Buildkite organization
- **Pipeline**: Select pipeline to trigger
- **Branch**: Git branch to build
- **Commit**: Git commit SHA (optional, defaults to HEAD)
- **Message**: Optional build message
- **Environment Variables**: Optional env vars for the build
- **Metadata**: Optional metadata for the build

## Output Channels

- **Passed**: Build completed successfully
- **Failed**: Build failed, was cancelled, or was blocked

## Polling Details

	- **Frequency**: Controlled by the PollInterval constant (currently 30 seconds)
	- **API**: Uses the Buildkite GetBuild endpoint to check build status
	- **Completion Detection**: Checks build state and emits to appropriate channel
	- **Manual Finish**: Manual action available to force completion if needed
	`
}
