package messages

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const PassedOutputChannel = "passed"
const FailedOutputChannel = "failed"

type RunApp struct{}

func init() {
	registry.RegisterAction("runApp", &RunApp{})
}

type RunAppConfiguration struct {
	App        string         `json:"app" mapstructure:"app"`
	Node       string         `json:"node" mapstructure:"node"`
	Parameters map[string]any `json:"parameters" mapstructure:"parameters"`
}

type runAppExecutionMetadata struct {
	Run *RunMetadata `json:"run" mapstructure:"run"`
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
	return "Run another SuperPlane app and wait for its run to finish"
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
		{Name: PassedOutputChannel, Label: "Passed"},
		{Name: FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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
			Name:        "parameters",
			Label:       "Parameters",
			Description: "The run parameters to pass to the target app",
			Type:        configuration.FieldTypeRunParameters,
			Required:    true,
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

	run, err := ctx.Runs.Create(core.RunCreationParams{
		App:   nodeMetadata.App.ID,
		Node:  nodeMetadata.Node.ID,
		Input: config.Parameters,
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

	return ctx.Metadata.Set(runAppExecutionMetadata{
		Run: &RunMetadata{
			ID: run.ID.String(),
		},
	})
}

func (c *RunApp) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "onRunFinished", Type: core.HookTypeInternal},
	}
}

func (c *RunApp) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "onRunFinished":
		return c.handleRunFinished(ctx)
	default:
		return fmt.Errorf("run app: unknown hook %s", ctx.Name)
	}
}

func (c *RunApp) handleRunFinished(ctx core.ActionHookContext) error {
	callback, err := core.DecodeRunFinishedCallback(ctx.Parameters)
	if err != nil {
		return fmt.Errorf("run app: decode run finished callback: %w", err)
	}

	executionMetadata := runAppExecutionMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &executionMetadata)
	if err != nil {
		return fmt.Errorf("run app: decode execution metadata: %w", err)
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

		return ctx.ExecutionState.Emit(PassedOutputChannel, "app.invocation.passed", []any{
			map[string]any{
				"run": map[string]any{
					"id":     callback.Run.ID.String(),
					"result": callback.Run.Result,
				},
			},
		})
	}

	errMessage := ""
	if callback.Run.Error != nil {
		errMessage = *callback.Run.Error
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

	return ctx.ExecutionState.Emit(FailedOutputChannel, "app.invocation.failed", []any{
		map[string]any{
			"run": map[string]any{
				"id":     callback.Run.ID.String(),
				"result": callback.Run.Result,
				"error":  errMessage,
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
