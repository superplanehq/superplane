package manual

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ActionRun = "run"

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
	return `The Manual Run trigger allows you to start workflow executions manually from the SuperPlane UI.

## Use Cases

- **Testing workflows**: Manually trigger workflows during development and testing
- **One-off tasks**: Run workflows on-demand for specific operations
- **Debugging**: Manually execute workflows to debug issues
- **Ad-hoc processing**: Process data when needed without automation

## How It Works

1. Add the Manual Run trigger as the starting node of your workflow
2. Click the "Run" button in the workflow UI to start an execution
3. The workflow begins immediately with empty event data

## Configuration

The Manual Run trigger requires no configuration. It's ready to use immediately after being added to a workflow.

## Event Data

Manual runs start with an empty event payload. You can use this as a starting point and add data through subsequent components.`
}

func (s *Start) Icon() string {
	return "play"
}

func (s *Start) Color() string {
	return "purple"
}

func (s *Start) Configuration() []configuration.Field {
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
								Name:     "payload",
								Label:    "Payload",
								Type:     configuration.FieldTypeObject,
								Required: true,
							},
						},
					},
				},
			},
			Default: []map[string]any{
				{
					"name":    "Hello World",
					"payload": map[string]any{"message": "Hello, World!"},
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

func (s *Start) Actions() []core.Action {
	return []core.Action{
		{
			Name:           ActionRun,
			Description:    "Start a manual run using a configured template",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:     "template",
					Label:    "Template",
					Type:     configuration.FieldTypeString,
					Required: true,
				},
				{
					Name:  "payload",
					Label: "Payload Override",
					Type:  configuration.FieldTypeObject,
				},
			},
		},
	}
}

func (s *Start) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case ActionRun:
		return s.run(ctx)
	default:
		return nil, fmt.Errorf("action %s not supported", ctx.Name)
	}
}

func (s *Start) run(ctx core.TriggerActionContext) (map[string]any, error) {
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
	var payload map[string]any

	for _, raw := range rawTemplates {
		tmpl, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tmpl["name"].(string)
		names = append(names, name)
		if name == templateName {
			payload, _ = tmpl["payload"].(map[string]any)
		}
	}

	if payload == nil && !containsName(names, templateName) {
		return nil, fmt.Errorf("template %q not found; available: %s", templateName, strings.Join(names, ", "))
	}

	if override, ok := ctx.Parameters["payload"].(map[string]any); ok {
		payload = override
	}

	if err := ctx.Events.Emit("manual.run", payload); err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return map[string]any{"template": templateName}, nil
}

func containsName(names []string, target string) bool {
	for _, n := range names {
		if n == target {
			return true
		}
	}
	return false
}

func (s *Start) Cleanup(ctx core.TriggerContext) error {
	return nil
}
