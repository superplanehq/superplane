package messages

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

type OnRun struct{}

type OnRunConfiguration struct {
	Parameters []configuration.Field `json:"parameters"`
}

func init() {
	registry.RegisterTrigger("onRun", &OnRun{})
}

func (c *OnRun) Name() string {
	return "onRun"
}

func (c *OnRun) Label() string {
	return "On Run"
}

func (c *OnRun) Description() string {
	return "Handle runs started from another app"
}

func (c *OnRun) Color() string {
	return "gray"
}

func (c *OnRun) Icon() string {
	return "play"
}

func (c *OnRun) Documentation() string {
	return ""
}

func (c *OnRun) ExampleData() map[string]any {
	return map[string]any{
		"timestamp": "2026-07-19T12:00:00Z",
		"type":      "app.invocation",
		"data": map[string]any{
			"app": map[string]any{
				"id":   "123",
				"name": "Source App",
			},
			"parameters": map[string]any{
				"parameter": "hello",
			},
		},
	}
}

func (c *OnRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "parameters",
			Label:       "Parameters",
			Description: "Parameters to receive when another app runs this app",
			Type:        configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "type",
								Label:       "Type",
								Description: "The type of the parameter",
								Type:        configuration.FieldTypeSelect,
								Required:    true,
								Default:     "string",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{
												Label: "String",
												Value: configuration.FieldTypeString,
											},
											{
												Label: "Number",
												Value: configuration.FieldTypeNumber,
											},
											{
												Label: "Boolean",
												Value: configuration.FieldTypeBool,
											},
										},
									},
								},
							},
							{
								Name:        "name",
								Label:       "Name",
								Description: "The name of the parameter",
								Type:        configuration.FieldTypeString,
								Required:    true,
								TypeOptions: &configuration.TypeOptions{
									String: &configuration.StringTypeOptions{
										AllowExpressions: new(bool),
									},
								},
							},
							{
								Name:        "required",
								Label:       "Required",
								Description: "Whether the parameter is required",
								Type:        configuration.FieldTypeBool,
								Default:     false,
							},
							{
								Name:        "label",
								Label:       "Label",
								Description: "The label of the parameter",
								Type:        configuration.FieldTypeString,
								Togglable:   true,
								TypeOptions: &configuration.TypeOptions{
									String: &configuration.StringTypeOptions{
										AllowExpressions: new(bool),
									},
								},
							},
							{
								Name:        "description",
								Label:       "Description",
								Description: "The description of the parameter",
								Type:        configuration.FieldTypeText,
								Togglable:   true,
							},
							{
								Name:        "default",
								Label:       "Default",
								Description: "The default value of the parameter",
								Type:        configuration.FieldTypeString,
								Togglable:   true,
								TypeOptions: &configuration.TypeOptions{
									String: &configuration.StringTypeOptions{
										AllowExpressions: new(bool),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *OnRun) Setup(ctx core.TriggerContext) error {
	return nil
}

func (c *OnRun) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "onMessage", Type: core.HookTypeInternal},
	}
}

func (c *OnRun) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	switch ctx.Name {
	case "onMessage":
		return c.handleMessage(ctx)
	default:
		return nil, fmt.Errorf("on run: unknown hook %s", ctx.Name)
	}
}

func (c *OnRun) handleMessage(ctx core.TriggerHookContext) (map[string]any, error) {
	config := OnRunConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("on run: decode configuration: %w", err)
	}

	parameters, ok := ctx.Parameters["parameters"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("on run: parameters not found")
	}

	sourceApp, ok := ctx.Parameters["app"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("on run: source app not found")
	}

	if len(config.Parameters) > 0 {
		err = configuration.ValidateConfiguration(config.Parameters, parameters)
		if err != nil {
			return nil, fmt.Errorf("on run: validate configuration: %w", err)
		}
	}

	err = ctx.Events.Emit("app.invocation", map[string]any{
		"app":        sourceApp,
		"parameters": parameters,
	})
	if err != nil {
		return nil, fmt.Errorf("on run: emit event: %w", err)
	}

	return nil, nil
}

func (c *OnRun) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *OnRun) Cleanup(ctx core.TriggerContext) error {
	return nil
}
