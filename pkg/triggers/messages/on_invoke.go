package messages

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

// TODO: call this just 'run'?
type OnInvoke struct{}

type OnInvokeConfiguration struct {
	Parameters []configuration.Field `json:"parameters"`
}

func init() {
	registry.RegisterTrigger("onInvoke", &OnInvoke{})
}

func (c *OnInvoke) Name() string {
	return "onInvoke"
}

func (c *OnInvoke) Label() string {
	return "On Invoke"
}

func (c *OnInvoke) Description() string {
	return "Handle invocations"
}

func (c *OnInvoke) Color() string {
	return "gray"
}

func (c *OnInvoke) Icon() string {
	return "play"
}

func (c *OnInvoke) Documentation() string {
	return ""
}

func (c *OnInvoke) ExampleData() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"id":   "123",
			"name": "Caller App",
		},
		"node": map[string]any{
			"id":   "invoke",
			"name": "Invoke App",
		},
		"payload": map[string]any{
			"message": "Hello, World!",
		},
	}
}

func (c *OnInvoke) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "parameters",
			Label:       "Parameters",
			Description: "Parameters to receive as part of the invocation",
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
											// TODO: support other types
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
								Name:        "label",
								Label:       "Label",
								Description: "The label of the parameter",
								Type:        configuration.FieldTypeString,
								Required:    true,
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
							},
							{
								Name:        "required",
								Label:       "Required",
								Description: "Whether the parameter is required",
								Type:        configuration.FieldTypeBool,
							},
							{
								Name:        "default",
								Label:       "Default",
								Description: "The default value of the parameter",
								Type:        configuration.FieldTypeString,
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

func (c *OnInvoke) Setup(ctx core.TriggerContext) error {
	// TODO: validate configuration for parameters is correct
	return nil
}

func (c *OnInvoke) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "onMessage", Type: core.HookTypeInternal},
	}
}

func (c *OnInvoke) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	switch ctx.Name {
	case "onMessage":
		return c.handleMessage(ctx)
	default:
		return nil, fmt.Errorf("on invoke: unknown hook %s", ctx.Name)
	}
}

func (c *OnInvoke) handleMessage(ctx core.TriggerHookContext) (map[string]any, error) {
	config := OnInvokeConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("on invoke: decode configuration: %w", err)
	}

	if len(config.Parameters) > 0 {
		err = configuration.ValidateConfiguration(config.Parameters, ctx.Parameters)
		if err != nil {
			return nil, fmt.Errorf("on invoke: validate configuration: %w", err)
		}
	}

	err = ctx.Events.Emit("app.invocation", ctx.Parameters)
	if err != nil {
		return nil, fmt.Errorf("on invoke: emit event: %w", err)
	}

	return nil, nil
}

func (c *OnInvoke) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *OnInvoke) Cleanup(ctx core.TriggerContext) error {
	return nil
}
