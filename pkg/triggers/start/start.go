package manual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const HookRun = "run"

func init() {
	registry.RegisterTrigger("start", &Start{})
}

type Start struct{}

func (s *Start) Name() string {
	return "start"
}

func (s *Start) Label() string {
	return "Manual Run"
}

func (s *Start) Description() string {
	return "Start a new execution chain manually"
}

func (s *Start) Documentation() string {
	return `The Manual Run trigger allows you to start workflow executions manually from the SuperPlane UI or CLI.

## Use Cases

- **Testing workflows**: Manually trigger workflows during development and testing
- **One-off tasks**: Run workflows on-demand for specific operations
- **Debugging**: Manually execute workflows to debug issues
- **Ad-hoc processing**: Process data when needed without automation

## How It Works

1. Add the Manual Run trigger as the starting node of your workflow
2. Configure one or more templates, each with a name, default payload, and optional parameters
3. Click the "Run" button in the workflow UI, or invoke the ` + "`run`" + ` hook via the API/CLI to start an execution
4. The workflow begins immediately with the configured payload for the selected template

## Configuration

Each Manual Run trigger exposes a list of templates. A template has:

- ` + "`name`" + ` (required): a label used as the run target (and the event channel)
- ` + "`parameters`" + ` (optional): a list of typed parameters exposed to payload expressions as ` + "`parameters[\"name\"]`" + ` and used by the run form
- ` + "`payload`" + ` (required): a default JSON object emitted when the template is used. Supports expressions such as ` + "`{{ now() }}`" + ` and ` + "`{{ parameters[\"my parameter\"] }}`" + ` in JSON values.

Each parameter has a ` + "`name`" + ` (plain text), required ` + "`type`" + ` (` + "`string`" + `, ` + "`text`" + `, ` + "`number`" + `, ` + "`boolean`" + `, or ` + "`select`" + `), an optional ` + "`title`" + ` for the run form label (defaults to ` + "`name`" + ` when unset), and an optional default (` + "`defaultString`" + `, ` + "`defaultNumber`" + `, or ` + "`defaultBoolean`" + `) whose editor matches the selected type. ` + "`string`" + ` is a single-line input and ` + "`text`" + ` renders a multi-line textarea; both use ` + "`defaultString`" + `. Select parameters also require an ` + "`options`" + ` list of ` + "`label`" + ` / ` + "`value`" + ` pairs; run-time values use the option ` + "`value`" + ` strings.

## Event Data

Manual runs emit an event with type ` + "`manual.run`" + `. The data is the selected template's payload after expression resolution.`
}

func (s *Start) Icon() string {
	return "play"
}

func (s *Start) Color() string {
	return "purple"
}

func (s *Start) Configuration() []configuration.Field {
	disallowExpressions := false

	return []configuration.Field{
		{
			Name:  "templates",
			Label: "Templates",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Template",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Template Name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:  "parameters",
								Label: "Parameters",
								Type:  configuration.FieldTypeList,
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel:   "Parameter",
										Accordion:   true,
										Reorderable: true,
										ItemDefinition: &configuration.ListItemDefinition{
											Type: configuration.FieldTypeObject,
											Schema: []configuration.Field{
												{
													Name:     "name",
													Label:    "Name",
													Type:     configuration.FieldTypeString,
													Required: true,
													TypeOptions: &configuration.TypeOptions{
														String: &configuration.StringTypeOptions{
															AllowExpressions: &disallowExpressions,
														},
													},
												},
												{
													Name:     "type",
													Label:    "Type",
													Type:     configuration.FieldTypeSelect,
													Required: true,
													Default:  configuration.FieldTypeString,
													TypeOptions: &configuration.TypeOptions{
														Select: &configuration.SelectTypeOptions{
															Options: []configuration.FieldOption{
																{Label: "String", Value: configuration.FieldTypeString},
																{Label: "Text (multi-line)", Value: configuration.FieldTypeText},
																{Label: "Number", Value: configuration.FieldTypeNumber},
																{Label: "Boolean", Value: configuration.FieldTypeBool},
																{Label: "Select", Value: configuration.FieldTypeSelect},
															},
														},
													},
												},
												{
													Name:  "options",
													Label: "Options",
													Type:  configuration.FieldTypeList,
													RequiredConditions: []configuration.RequiredCondition{
														{Field: "type", Values: []string{configuration.FieldTypeSelect}},
													},
													VisibilityConditions: []configuration.VisibilityCondition{
														{Field: "type", Values: []string{configuration.FieldTypeSelect}},
													},
													TypeOptions: &configuration.TypeOptions{
														List: &configuration.ListTypeOptions{
															ItemLabel: "Option",
															Accordion: true,
															ItemDefinition: &configuration.ListItemDefinition{
																Type: configuration.FieldTypeObject,
																Schema: []configuration.Field{
																	{
																		Name:     "label",
																		Label:    "Label",
																		Type:     configuration.FieldTypeString,
																		Required: true,
																		TypeOptions: &configuration.TypeOptions{
																			String: &configuration.StringTypeOptions{
																				AllowExpressions: &disallowExpressions,
																			},
																		},
																	},
																	{
																		Name:     "value",
																		Label:    "Value",
																		Type:     configuration.FieldTypeString,
																		Required: true,
																		TypeOptions: &configuration.TypeOptions{
																			String: &configuration.StringTypeOptions{
																				AllowExpressions: &disallowExpressions,
																			},
																		},
																	},
																},
															},
														},
													},
												},
												{
													Name:      "defaultString",
													Label:     "Default Value",
													Type:      configuration.FieldTypeString,
													Togglable: true,
													VisibilityConditions: []configuration.VisibilityCondition{
														{Field: "type", Values: []string{configuration.FieldTypeString, configuration.FieldTypeText, configuration.FieldTypeSelect}},
													},
													TypeOptions: &configuration.TypeOptions{
														String: &configuration.StringTypeOptions{
															AllowExpressions: &disallowExpressions,
														},
													},
												},
												{
													Name:      "defaultNumber",
													Label:     "Default Value",
													Type:      configuration.FieldTypeNumber,
													Togglable: true,
													VisibilityConditions: []configuration.VisibilityCondition{
														{Field: "type", Values: []string{configuration.FieldTypeNumber}},
													},
												},
												{
													Name:      "defaultBoolean",
													Label:     "Default Value",
													Type:      configuration.FieldTypeBool,
													Togglable: true,
													VisibilityConditions: []configuration.VisibilityCondition{
														{Field: "type", Values: []string{configuration.FieldTypeBool}},
													},
												},
												{
													Name:      "title",
													Label:     "Display Title",
													Togglable: true,
													Type:      configuration.FieldTypeString,
													TypeOptions: &configuration.TypeOptions{
														String: &configuration.StringTypeOptions{
															AllowExpressions: &disallowExpressions,
														},
													},
												},
												{
													Name:      "placeholder",
													Label:     "Input Placeholder",
													Togglable: true,
													Type:      configuration.FieldTypeString,
													VisibilityConditions: []configuration.VisibilityCondition{
														{Field: "type", Values: []string{configuration.FieldTypeString, configuration.FieldTypeText, configuration.FieldTypeNumber}},
													},
													TypeOptions: &configuration.TypeOptions{
														String: &configuration.StringTypeOptions{
															AllowExpressions: &disallowExpressions,
														},
													},
												},
											},
										},
									},
								},
							},
							{
								Name:        "payload",
								Label:       "Payload",
								Type:        configuration.FieldTypeObject,
								Required:    true,
								Description: "JSON object emitted when the template runs. Supports expressions such as {{ now() }} and {{ parameters[\"my parameter\"] }} in field values.",
								Placeholder: `{
  "message": "Hello, World!"
}`,
							},
						},
					},
				},
			},
			Default: []map[string]any{
				{
					"name":       "Hello World",
					"payload":    map[string]any{"message": "Hello, World!"},
					"parameters": []map[string]any{},
				},
			},
		},
	}
}

func (s *Start) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (s *Start) Setup(ctx core.TriggerContext) error {
	return nil
}

func (s *Start) Hooks() []core.Hook {
	return []core.Hook{
		{
			Type: core.HookTypeUser,
			Name: HookRun,
			Parameters: []configuration.Field{
				{
					Name:     "template",
					Label:    "Template",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
			},
		},
	}
}

func (s *Start) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	switch ctx.Name {
	case HookRun:
		return s.run(ctx)
	default:
		return nil, fmt.Errorf("hook %s not supported", ctx.Name)
	}
}

func (s *Start) run(ctx core.TriggerHookContext) (map[string]any, error) {
	templateName, _ := ctx.Parameters["template"].(string)
	if templateName == "" {
		return nil, fmt.Errorf("template parameter is required")
	}

	config, _ := ctx.Configuration.(map[string]any)
	rawTemplates, _ := config["templates"].([]any)
	if len(rawTemplates) == 0 {
		return nil, fmt.Errorf("no templates configured")
	}

	var names []string
	var payload any
	found := false

	for _, raw := range rawTemplates {
		tmpl, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tmpl["name"].(string)
		names = append(names, name)
		if name == templateName {
			payload = templatePayload(tmpl)
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("template %q not found; available: %s", templateName, strings.Join(names, ", "))
	}

	if payload == nil {
		return nil, fmt.Errorf("template %q has no payload", templateName)
	}

	resolvedPayload, err := payloadObject(payload)
	if err != nil {
		return nil, fmt.Errorf("template %q payload must be a JSON object: %w", templateName, err)
	}

	if err := ctx.Events.Emit("manual.run", resolvedPayload); err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return map[string]any{"template": templateName}, nil
}

func (s *Start) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func templatePayload(tmpl map[string]any) any {
	return tmpl["payload"]
}

func payloadObject(value any) (map[string]any, error) {
	switch payload := value.(type) {
	case map[string]any:
		return payload, nil
	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
			return nil, err
		}
		if parsed == nil {
			return nil, fmt.Errorf("empty payload")
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported payload type")
	}
}
