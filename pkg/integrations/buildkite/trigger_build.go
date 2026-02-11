package buildkite

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
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
	Pipeline string `json:"pipeline"`
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

type KeyValuePair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TriggerBuildSpec struct {
	Pipeline string         `json:"pipeline"`
	Branch   string         `json:"branch"`
	Commit   string         `json:"commit"`
	Message  string         `json:"message,omitempty"`
	Env      []KeyValuePair `json:"env,omitempty" mapstructure:"env"`
	Metadata []KeyValuePair `json:"meta_data,omitempty" mapstructure:"meta_data"`
}

func (r *TriggerBuild) Name() string {
	return "buildkite.triggerBuild"
}

func (r *TriggerBuild) Label() string {
	return "Trigger Build"
}

func (r *TriggerBuild) Description() string {
	return "Trigger a Buildkite build and wait for completion."
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
			Name:     "pipeline",
			Label:    "Pipeline",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pipeline",
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

	if config.Pipeline == "" {
		return fmt.Errorf("pipeline is required")
	}

	metadata := TriggerBuildNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Pipeline == config.Pipeline {
		return nil
	}

	err = ctx.Metadata.Set(TriggerBuildNodeMetadata{
		Pipeline: config.Pipeline,
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

	orgConfig, err := ctx.Integration.GetConfig("organization")
	if err != nil {
		return fmt.Errorf("failed to get organization from integration config: %w", err)
	}
	orgSlug, err := extractOrgSlug(string(orgConfig))
	if err != nil {
		return fmt.Errorf("failed to extract organization slug: %w", err)
	}

	envMap := make(map[string]string, len(spec.Env))
	for _, e := range spec.Env {
		envMap[e.Name] = e.Value
	}

	metadataMap := make(map[string]string, len(spec.Metadata)+2)
	for _, m := range spec.Metadata {
		metadataMap[m.Name] = m.Value
	}
	metadataMap["superplane_execution_id"] = ctx.ID.String()
	metadataMap["superplane_workflow_id"] = ctx.WorkflowID

	buildReq := CreateBuildRequest{
		Commit:   commit,
		Branch:   spec.Branch,
		Message:  spec.Message,
		Env:      envMap,
		Metadata: metadataMap,
	}

	build, err := client.CreateBuild(orgSlug, spec.Pipeline, buildReq)
	if err != nil {
		return fmt.Errorf("error creating build: %v", err)
	}

	err = ctx.Metadata.Set(TriggerBuildExecutionMetadata{
		BuildID:      build.ID,
		BuildNumber:  build.Number,
		WebURL:       build.WebURL,
		Organization: orgSlug,
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
				"id":          targetBuild.ID,
				"number":      targetBuild.Number,
				"state":       targetBuild.State,
				"web_url":     targetBuild.WebURL,
				"commit":      targetBuild.Commit,
				"branch":      targetBuild.Branch,
				"message":     targetBuild.Message,
				"blocked":     targetBuild.Blocked,
				"started_at":  targetBuild.StartedAt,
				"finished_at": targetBuild.FinishedAt,
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

		return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
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

1. Triggers a build via Buildkite API
2. Monitors build status via polling
3. Emits to success/failure channels when build finishes

## Configuration

- **Pipeline**: Select pipeline to trigger
- **Branch**: Git branch to build
- **Commit**: Git commit SHA (optional, defaults to HEAD)
- **Message**: Optional build message
- **Environment Variables**: Optional env vars for the build
- **Metadata**: Optional metadata for the build

## Output Channels

- **Passed**: Build completed successfully
- **Failed**: Build failed, was cancelled, or was blocked`
}
