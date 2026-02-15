package triggers

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	name := ctx.Args[0]
	var trigger openapi_client.TriggersTrigger

	integrationName, triggerName, scoped := core.ParseIntegrationScopedName(name)
	if scoped {
		integration, err := core.FindIntegrationDefinition(ctx, integrationName)
		if err != nil {
			return err
		}

		resolvedTrigger, err := findTrigger(integration, triggerName)
		if err != nil {
			return err
		}
		trigger = resolvedTrigger
	} else {
		response, _, err := ctx.API.TriggerAPI.TriggersDescribeTrigger(ctx.Context, name).Execute()
		if err != nil {
			return err
		}
		trigger = response.GetTrigger()
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, _ = fmt.Fprintf(stdout, "Name: %s\n", trigger.GetName())
			_, _ = fmt.Fprintf(stdout, "Label: %s\n", trigger.GetLabel())
			_, err := fmt.Fprintf(stdout, "Description: %s\n", trigger.GetDescription())
			return err
		})
	}

	return ctx.Renderer.Render(trigger)
}

func findTrigger(integration openapi_client.IntegrationsIntegrationDefinition, name string) (openapi_client.TriggersTrigger, error) {
	for _, trigger := range integration.GetTriggers() {
		triggerName := trigger.GetName()
		if triggerName == name || triggerName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return trigger, nil
		}
	}

	return openapi_client.TriggersTrigger{}, fmt.Errorf("trigger %q not found in integration %q", name, integration.GetName())
}
