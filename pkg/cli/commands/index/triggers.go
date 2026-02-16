package index

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func newTriggersCommand(options core.BindOptions) *cobra.Command {
	var from string
	var name string

	cmd := &cobra.Command{
		Use:   "triggers",
		Short: "List or describe available triggers",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&from, "from", "", "integration definition name")
	cmd.Flags().StringVar(&name, "name", "", "trigger name")
	core.Bind(cmd, &triggersCommand{from: &from, name: &name}, options)

	return cmd
}

type triggersCommand struct {
	from *string
	name *string
}

func (c *triggersCommand) Execute(ctx core.CommandContext) error {
	name := strings.TrimSpace(*c.name)
	from := strings.TrimSpace(*c.from)

	if name != "" {
		return c.getTriggerByName(ctx, name)
	}

	triggers, err := c.listTriggers(ctx, from)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(triggers)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, trigger := range triggers {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", trigger.GetName(), trigger.GetLabel(), trigger.GetDescription())
		}
		return writer.Flush()
	})
}

func (c *triggersCommand) getTriggerByName(ctx core.CommandContext, name string) error {
	trigger, err := c.findTriggerByName(ctx, name)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(trigger)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", trigger.GetName())
		_, _ = fmt.Fprintf(stdout, "Label: %s\n", trigger.GetLabel())
		_, err := fmt.Fprintf(stdout, "Description: %s\n", trigger.GetDescription())
		return err
	})
}

func (c *triggersCommand) listTriggers(ctx core.CommandContext, from string) ([]openapi_client.TriggersTrigger, error) {
	//
	// if --from is used, we grab the triggers from the integration
	//
	if from != "" {
		integration, err := core.FindIntegrationDefinition(ctx, from)
		if err != nil {
			return nil, err
		}

		return integration.GetTriggers(), nil
	}

	//
	// Otherwise, we list core triggers.
	//
	response, _, err := ctx.API.TriggerAPI.TriggersListTriggers(ctx.Context).Execute()
	if err != nil {
		return nil, err
	}

	return response.GetTriggers(), nil
}

func (c *triggersCommand) findTriggerByName(ctx core.CommandContext, name string) (openapi_client.TriggersTrigger, error) {
	integrationName, triggerName, scoped := core.ParseIntegrationScopedName(name)
	if scoped {
		integration, err := core.FindIntegrationDefinition(ctx, integrationName)
		if err != nil {
			return openapi_client.TriggersTrigger{}, err
		}
		return findIntegrationTrigger(integration, triggerName)
	}

	response, _, err := ctx.API.TriggerAPI.TriggersDescribeTrigger(ctx.Context, name).Execute()
	if err != nil {
		return openapi_client.TriggersTrigger{}, err
	}
	return response.GetTrigger(), nil
}

func findIntegrationTrigger(
	integration openapi_client.IntegrationsIntegrationDefinition,
	name string,
) (openapi_client.TriggersTrigger, error) {
	for _, trigger := range integration.GetTriggers() {
		triggerName := trigger.GetName()
		if triggerName == name || triggerName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return trigger, nil
		}
	}

	return openapi_client.TriggersTrigger{}, fmt.Errorf("trigger %q not found in integration %q", name, integration.GetName())
}
