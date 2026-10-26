package runs

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	RunAppPassedOutputChannel   = "passed"
	RunAppFailedOutputChannel   = "failed"
	RunAppActionRunTimeout      = "onRunTimeout"
	defaultRunAppTimeoutSeconds = 3600
)

type RunApp struct{}

func init() {
	registry.RegisterAction("runApp", &RunApp{})
}

type RunAppConfiguration struct {
	App        string         `json:"app" mapstructure:"app"`
	Node       string         `json:"node" mapstructure:"node"`
	Parameters map[string]any `json:"parameters" mapstructure:"parameters"`
	Timeout    *int           `json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
}

func (c *RunAppConfiguration) TimeoutDuration() time.Duration {
	if c.Timeout == nil || *c.Timeout <= 0 {
		return time.Duration(defaultRunAppTimeoutSeconds) * time.Second
	}

	return time.Duration(*c.Timeout) * time.Second
}

type runAppExecutionMetadata struct {
	Run                 *RunMetadata `json:"run" mapstructure:"run"`
	QueueTimeoutVersion int          `json:"queueTimeoutVersion,omitempty" mapstructure:"queueTimeoutVersion"`
	TimeoutStartedAt    string       `json:"timeoutStartedAt,omitempty" mapstructure:"timeoutStartedAt"`
}

type RunMetadata struct {
	ID     string  `json:"id" mapstructure:"id"`
	Result string  `json:"result" mapstructure:"result"`
	Error  *string `json:"error,omitempty" mapstructure:"error,omitempty"`
}

type RunAppMetadata struct {
	App  *AppMetadata        `json:"app" mapstructure:"app"`
	Node *CanvasNodeMetadata `json:"node" mapstructure:"node"`
}

type AppMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

type CanvasNodeMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (c *RunApp) Name() string {
	return "runApp"
}

func (c *RunApp) Label() string {
	return "Run App"
}

func (c *RunApp) Color() string {
	return "gray"
}

func (c *RunApp) Icon() string {
	return "play"
}

func (c *RunApp) Documentation() string {
	return `Run another SuperPlane app and wait for its run to finish.

## Configuration

- **App**: The SuperPlane app to invoke
- **Node**: The On Run trigger in the target app
- **Parameters**: Values passed to the target app's On Run trigger
- **Timeout**: Maximum time in seconds for the child run (default 3600). Runner queue waits in the child and its descendants pause this timeout. Provisioning, preparation, approvals, and automation slot waits count toward the timeout. When the timeout expires, SuperPlane cancels the child and uses the Failed output channel.`
}

func (c *RunApp) Description() string {
	return "Run another SuperPlane app and wait for its run to finish"
}

func (c *RunApp) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-07-19T12:00:00Z",
		"type":      "app.invocation.passed",
		"data": map[string]any{
			"run": map[string]any{
				"id":     "123",
				"result": core.RunResultPassed,
			},
		},
	}
}

func (c *RunApp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: RunAppPassedOutputChannel, Label: "Passed"},
		{Name: RunAppFailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunApp) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "app",
			Label:       "App",
			Description: "The SuperPlane app to run",
			Type:        configuration.FieldTypeApp,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				App: &configuration.AppTypeOptions{
					AllowSelf: true,
				},
			},
		},
		{
			Name:        "node",
			Label:       "Node",
			Description: "The On Run trigger in the target app",
			Type:        configuration.FieldTypeAppCanvasNode,
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "app",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AppCanvasNode: &configuration.AppCanvasNodeTypeOptions{
					NodeTypes:      []string{"trigger"},
					ComponentTypes: []string{"onRun"},
					Parameters: []configuration.ParameterRef{
						{
							Name: "app",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "app",
							},
						},
					},
				},
			},
		},
		{
			Name:  "parameters",
			Label: "Parameters",
			Type:  configuration.FieldTypeRunParameters,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "app",
					Values: []string{"*"},
				},
				{
					Field:  "node",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeNumber,
			Description: "Maximum time to wait for the child run in seconds (default 3600)",
			Required:    false,
			Default:     "3600",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
				},
			},
		},
	}
}

func (c *RunApp) Setup(ctx core.SetupContext) error {
	config := RunAppConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.App == "" {
		return fmt.Errorf("app is required")
	}

	if config.Node == "" {
		return fmt.Errorf("node is required")
	}

	app, err := ctx.Apps.Get(config.App)
	if err != nil {
		return fmt.Errorf("failed to get app node: %w", err)
	}

	node, err := ctx.Apps.GetNode(app.ID, config.Node)
	if err != nil {
		return fmt.Errorf("failed to get app node: %w", err)
	}

	metadata := RunAppMetadata{
		App: &AppMetadata{
			ID:   app.ID,
			Name: app.Name,
		},
		Node: &CanvasNodeMetadata{
			ID:   node.ID,
			Name: node.Name,
		},
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("run app: set metadata: %w", err)
	}

	return nil
}

func (c *RunApp) Execute(ctx core.ExecutionContext) error {
	config := RunAppConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("run app: decode configuration: %w", err)
	}

	nodeMetadata := RunAppMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("run app: decode configuration: %w", err)
	}

	if nodeMetadata.App == nil || nodeMetadata.Node == nil {
		return fmt.Errorf("run app: metadata is required")
	}

	input := map[string]any{
		"app": map[string]any{
			"id":   ctx.WorkflowID,
			"name": ctx.CanvasName,
		},
		"parameters": config.Parameters,
	}

	run, err := ctx.Runs.Create(core.RunCreationParams{
		App:   nodeMetadata.App.ID,
		Node:  nodeMetadata.Node.ID,
		Input: input,
		Callbacks: []core.RunCallback{
			{
				When: core.RunCallbackWhenPending,
				On:   core.RunCallbackOnEntry,
				Hook: "onMessage",
			},
			{
				When: core.RunCallbackWhenFinished,
				On:   core.RunCallbackOnParent,
				Hook: "onRunFinished",
			},
		},
	})

	if err != nil {
		return fmt.Errorf("run app: create run: %w", err)
	}

	executionMetadata := runAppExecutionMetadata{
		QueueTimeoutVersion: 1,
		TimeoutStartedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Run: &RunMetadata{
			ID: run.ID.String(),
		},
	}

	if err := ctx.Requests.ScheduleActionCall(RunAppActionRunTimeout, map[string]any{}, config.TimeoutDuration()); err != nil {
		return fmt.Errorf("run app: schedule timeout: %w", err)
	}

	return ctx.Metadata.Set(executionMetadata)
}

func (c *RunApp) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "onRunFinished", Type: core.HookTypeInternal},
		{Name: RunAppActionRunTimeout, Type: core.HookTypeInternal},
	}
}

func (c *RunApp) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "onRunFinished":
		return c.handleRunFinished(ctx)
	case RunAppActionRunTimeout:
		return c.handleRunTimeout(ctx)
	default:
		return fmt.Errorf("run app: unknown hook %s", ctx.Name)
	}
}

func (c *RunApp) handleRunTimeout(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}
	var metadata runAppExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("run app: decode timeout metadata: %w", err)
	}
	// Invocations created before queue history was recorded retain their policy.
	if metadata.QueueTimeoutVersion == 1 && metadata.Run != nil {
		started, err := time.Parse(time.RFC3339Nano, metadata.TimeoutStartedAt)
		if err != nil {
			return fmt.Errorf("run app: invalid timeout start: %w", err)
		}
		now := time.Now()
		wait, err := ctx.Runs.QueueWait(metadata.Run.ID, started, now)
		if err != nil {
			return fmt.Errorf("run app: measure runner queue wait: %w", err)
		}
		if wait.Finished {
			return nil
		}
		var config RunAppConfiguration
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return fmt.Errorf("run app: decode timeout configuration: %w", err)
		}
		remaining := config.TimeoutDuration() - now.Sub(started) + wait.Duration
		if remaining > 0 {
			if wait.Waiting {
				remaining = min(remaining, 15*time.Second)
			}
			return ctx.Requests.ScheduleActionCall(RunAppActionRunTimeout, map[string]any{}, remaining)
		}
	}

	return ctx.Runs.Cancel()
}

func (c *RunApp) handleRunFinished(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	callback, err := core.DecodeRunFinishedCallback(ctx.Parameters)
	if err != nil {
		return fmt.Errorf("run app: decode run finished callback: %w", err)
	}

	if callback.Run.Result == core.RunResultPassed {
		err = ctx.Metadata.Set(runAppExecutionMetadata{
			Run: &RunMetadata{
				ID:     callback.Run.ID.String(),
				Result: callback.Run.Result,
			},
		})

		if err != nil {
			return fmt.Errorf("run app: set execution metadata: %w", err)
		}

		return ctx.ExecutionState.Emit(RunAppPassedOutputChannel, "app.invocation.passed", []any{
			map[string]any{
				"run": map[string]any{
					"id":     callback.Run.ID.String(),
					"result": callback.Run.Result,
				},
			},
		})
	}

	errMessage := ""
	if len(callback.Run.Errors) > 0 {
		errMessage = callback.Run.Errors[0]
	}

	err = ctx.Metadata.Set(runAppExecutionMetadata{
		Run: &RunMetadata{
			ID:     callback.Run.ID.String(),
			Result: callback.Run.Result,
			Error:  &errMessage,
		},
	})

	if err != nil {
		return fmt.Errorf("run app: set execution metadata: %w", err)
	}

	return ctx.ExecutionState.Emit(RunAppFailedOutputChannel, "app.invocation.failed", []any{
		map[string]any{
			"run": map[string]any{
				"id":     callback.Run.ID.String(),
				"result": callback.Run.Result,
				"errors": callback.Run.Errors,
			},
		},
	})
}

func (c *RunApp) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *RunApp) Cancel(ctx core.ExecutionContext) error {
	return ctx.Runs.Cancel()
}

func (c *RunApp) Cleanup(ctx core.SetupContext) error {
	return nil
}
