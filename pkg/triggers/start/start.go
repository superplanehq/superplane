package manual

import (
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
2. Configure one or more templates, each with a name and a default payload
3. Click the "Run" button in the workflow UI, or invoke the ` + "`run`" + ` hook via the API/CLI to start an execution
4. The workflow begins immediately with either the configured payload or an override supplied at run time

## Configuration

Each Manual Run trigger exposes a list of templates. A template has:

- ` + "`name`" + ` (required): a label used as the run target (and the event channel)
- ` + "`payload`" + ` (required): a default JSON object emitted when the template is used

## Event Data

Manual runs emit an event with type ` + "`manual.run`" + `. The data is either the template's saved payload or the override passed in the run request.`
}

func (s *Start) Icon() string {
	return "play"
}

func (s *Start) Color() string {
	return "purple"
}

func (s *Start) DefaultRunTitle() string {
	return "Event emitted by trigger"
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
				{
					Name:  "payload",
					Label: "Payload Override",
					Type:  configuration.FieldTypeObject,
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
	var payload map[string]any
	found := false

	for _, raw := range rawTemplates {
		tmpl, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tmpl["name"].(string)
		names = append(names, name)
		if name == templateName {
			payload, _ = tmpl["payload"].(map[string]any)
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("template %q not found; available: %s", templateName, strings.Join(names, ", "))
	}

	if override, ok := ctx.Parameters["payload"].(map[string]any); ok {
		payload = override
	}

	if payload == nil {
		return nil, fmt.Errorf("template %q has no payload and no override was provided", templateName)
	}

	if err := ctx.Events.Emit("manual.run", payload); err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return map[string]any{"template": templateName}, nil
}

func (s *Start) Cleanup(ctx core.TriggerContext) error {
	return nil
}
