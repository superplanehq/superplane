package messages

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type InvokeApp struct{}

func init() {
	registry.RegisterAction("invokeApp", &InvokeApp{})
}

type InvokeAppConfiguration struct {
	App        string         `json:"app" mapstructure:"app"`
	Node       string         `json:"node" mapstructure:"node"`
	Parameters map[string]any `json:"parameters" mapstructure:"parameters"`
}

type InvokeAppMetadata struct {
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

func (c *InvokeApp) Name() string {
	return "invokeApp"
}

func (c *InvokeApp) Label() string {
	return "Invoke App"
}

func (c *InvokeApp) Color() string {
	return "gray"
}

func (c *InvokeApp) Icon() string {
	return "play"
}

func (c *InvokeApp) Documentation() string {
	return "Invoke another SuperPlane app and wait for its run to finish"
}

func (c *InvokeApp) Description() string {
	return "Invoke another SuperPlane app and wait for its run to finish"
}

func (c *InvokeApp) ExampleOutput() map[string]any {
	return map[string]any{
		"run": map[string]any{
			"id":     "123",
			"result": core.RunResultPassed,
		},
	}
}

func (c *InvokeApp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *InvokeApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *InvokeApp) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "app",
			Label:       "App",
			Description: "The SuperPlane app to invoke",
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
			Description: "The node to invoke",
			Type:        configuration.FieldTypeAppCanvasNode,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				AppCanvasNode: &configuration.AppCanvasNodeTypeOptions{
					NodeTypes:      []string{"trigger"},
					ComponentTypes: []string{"onInvoke"},
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
			Description: "The invocation parameters to use",
			Type:        configuration.FieldTypeInvocationParameters,
			Required:    true,
		},
	}
}

func (c *InvokeApp) Setup(ctx core.SetupContext) error {
	config := InvokeAppConfiguration{}
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

	metadata := InvokeAppMetadata{
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
		return fmt.Errorf("invoke app: set metadata: %w", err)
	}

	return nil
}

func (c *InvokeApp) Execute(ctx core.ExecutionContext) error {
	config := InvokeAppConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("invoke app: decode configuration: %w", err)
	}

	nodeMetadata := InvokeAppMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("invoke app: decode configuration: %w", err)
	}

	if nodeMetadata.App == nil || nodeMetadata.Node == nil {
		return fmt.Errorf("invoke app: metadata is required")
	}

	_, err = ctx.Runs.Create(core.RunCreationParams{
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
		return fmt.Errorf("invoke app: create run: %w", err)
	}

	return nil
}

func (c *InvokeApp) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "onRunFinished", Type: core.HookTypeInternal},
	}
}

func (c *InvokeApp) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "onRunFinished":
		return c.handleRunFinished(ctx)
	default:
		return fmt.Errorf("invoke app: unknown hook %s", ctx.Name)
	}
}

func (c *InvokeApp) handleRunFinished(ctx core.ActionHookContext) error {
	callback, err := core.DecodeRunFinishedCallback(ctx.Parameters)
	if err != nil {
		return fmt.Errorf("invoke app: decode run finished callback: %w", err)
	}

	if callback.Run.Result == core.RunResultPassed {
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "app.invocation.passed", []any{
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

	return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, errMessage)
}

func (c *InvokeApp) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *InvokeApp) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *InvokeApp) Cleanup(ctx core.SetupContext) error {
	return nil
}
