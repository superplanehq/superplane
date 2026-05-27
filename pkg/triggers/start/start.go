package manual

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers/start/params"
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
4. Click **Run** on a template: if the payload has no parameters, the run starts immediately; otherwise a form collects parameterized values before starting
5. The workflow begins with the resolved payload (static values plus submitted parameter values)

## Configuration

Each Manual Run trigger exposes a list of templates. A template has:

- ` + "`name`" + ` (required): a label used as the run target (and the event channel)
- ` + "`payload`" + ` (required): a JSON object emitted when the template is used

### Parameterized payload values

String values in the payload may use ` + "`param(...)`" + ` placeholders. At run time the UI renders a typed form for those fields.

Example:

` + "`param(type:string, title:'Enter a machine name', default:'machine-1', required:false)`" + `

Supported types: ` + "`string`" + `, ` + "`number`" + `, ` + "`boolean`" + `, and ` + "`select`" + ` (with ` + "`values:'opt1|opt2'`" + `).

## Event Data

Manual runs emit an event with type ` + "`manual.run`" + `. The data is the template payload with all ` + "`param(...)`" + ` placeholders replaced by submitted values.`
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
	config, _ := ctx.Configuration.(map[string]any)
	rawTemplates, _ := config["templates"].([]any)
	for _, raw := range rawTemplates {
		tmpl, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tmpl["name"].(string)
		payload, ok := tmpl["payload"].(map[string]any)
		if !ok || payload == nil {
			continue
		}
		if _, err := params.ValidatePayload(payload); err != nil {
			if name == "" {
				return err
			}
			return fmt.Errorf("template %q: %w", name, err)
		}
	}
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

	if params.ContainsUnresolvedParams(payload) {
		return nil, fmt.Errorf("template %q payload still contains unresolved param(...) placeholders", templateName)
	}

	if err := ctx.Events.Emit("manual.run", payload); err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	return map[string]any{"template": templateName}, nil
}

func (s *Start) Cleanup(ctx core.TriggerContext) error {
	return nil
}
