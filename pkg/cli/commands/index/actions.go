package index

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func newActionsCommand(options core.BindOptions) *cobra.Command {
	var from string
	var name string
	var full bool

	cmd := &cobra.Command{
		Use:   "actions",
		Short: "List or describe available actions",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&from, "from", "", "integration definition name")
	cmd.Flags().StringVar(&name, "name", "", "action name")
	cmd.Flags().BoolVar(&full, "full", false, "show full output including all fields")
	core.Bind(cmd, &actionsCommand{from: &from, name: &name, full: &full}, options)

	return cmd
}

type actionsCommand struct {
	from *string
	name *string
	full *bool
}

func (c *actionsCommand) Execute(ctx core.CommandContext) error {
	name := strings.TrimSpace(*c.name)
	from := strings.TrimSpace(*c.from)

	if name != "" {
		return c.getActionByName(ctx, name)
	}

	actions, err := c.listActions(ctx, from)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		if c.full != nil && *c.full {
			return ctx.Renderer.Render(actions)
		}

		summary := make([]map[string]string, len(actions))
		for i, action := range actions {
			summary[i] = map[string]string{
				"name":        action.GetName(),
				"label":       action.GetLabel(),
				"description": action.GetDescription(),
			}
		}
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, action := range actions {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", action.GetName(), action.GetLabel(), action.GetDescription())
		}
		return writer.Flush()
	})
}

func (c *actionsCommand) getActionByName(ctx core.CommandContext, name string) error {
	action, err := c.findActionByName(ctx, name)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(action)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderActionText(stdout, action)
	})
}

func renderActionText(stdout io.Writer, action openapi_client.SuperplaneActionsAction) error {
	_, err := fmt.Fprintf(stdout, "Name: %s\n", action.GetName())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Label: %s\n", action.GetLabel())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Description: %s\n", action.GetDescription())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout)
	if err != nil {
		return err
	}

	err = renderActionOutputChannelsText(stdout, action.GetOutputChannels())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout)
	if err != nil {
		return err
	}

	err = renderConfigurationText(stdout, action.GetConfiguration())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout)
	if err != nil {
		return err
	}

	return renderExamplePayloadText(stdout, action.GetExampleOutput())
}

func renderActionOutputChannelsText(stdout io.Writer, channels []openapi_client.SuperplaneActionsOutputChannel) error {
	_, err := fmt.Fprintln(stdout, "Output Channels:")
	if err != nil {
		return err
	}

	if len(channels) == 0 {
		_, err = fmt.Fprintln(stdout, "  (none)")
		return err
	}

	for _, channel := range channels {
		channelName := channel.GetName()
		if channelName == "" {
			channelName = "(unnamed)"
		}

		channelLabel := channel.GetLabel()
		channelDescription := channel.GetDescription()

		if channelLabel != "" && channelLabel != channelName {
			channelName = fmt.Sprintf("%s (%s)", channelName, channelLabel)
		}

		if channelDescription == "" {
			_, err = fmt.Fprintf(stdout, "  - %s\n", channelName)
			if err != nil {
				return err
			}
			continue
		}

		_, err = fmt.Fprintf(stdout, "  - %s: %s\n", channelName, channelDescription)
		if err != nil {
			return err
		}
	}

	return nil
}

func renderConfigurationText(stdout io.Writer, configuration []openapi_client.ConfigurationField) error {
	_, err := fmt.Fprintln(stdout, "Configuration:")
	if err != nil {
		return err
	}

	if len(configuration) == 0 {
		_, err = fmt.Fprintln(stdout, "  (none)")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "  NAME\tTYPE\tREQUIRED\tDESCRIPTION")
	for _, field := range configuration {
		required := "no"
		if field.GetRequired() {
			required = "yes"
		}

		_, _ = fmt.Fprintf(writer, "  %s\t%s\t%s\t%s\n", field.GetName(), field.GetType(), required, field.GetDescription())
	}

	return writer.Flush()
}

func renderExamplePayloadText(stdout io.Writer, examplePayload map[string]interface{}) error {
	_, err := fmt.Fprintln(stdout, "Example Payload:")
	if err != nil {
		return err
	}

	if len(examplePayload) == 0 {
		_, err = fmt.Fprintln(stdout, "  (none)")
		return err
	}

	serializedPayload, err := json.MarshalIndent(examplePayload, "", "  ")
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(serializedPayload), "\n") {
		_, err = fmt.Fprintf(stdout, "  %s\n", line)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *actionsCommand) listActions(ctx core.CommandContext, from string) ([]openapi_client.SuperplaneActionsAction, error) {
	//
	// if --from is used, we grab the actions from the integration
	//
	if from != "" {
		integration, err := core.FindIntegrationDefinition(ctx, from)
		if err != nil {
			return nil, err
		}

		return integration.GetActions(), nil
	}

	//
	// Otherwise, we list core actions.
	//
	response, _, err := ctx.API.ActionAPI.ActionsListActions(ctx.Context).Execute()
	if err != nil {
		return nil, err
	}
	return response.GetActions(), nil
}

func (c *actionsCommand) findActionByName(ctx core.CommandContext, name string) (openapi_client.SuperplaneActionsAction, error) {
	integrationName, componentName, scoped := core.ParseIntegrationScopedName(name)
	if scoped {
		integration, err := core.FindIntegrationDefinition(ctx, integrationName)
		if err != nil {
			return openapi_client.SuperplaneActionsAction{}, fmt.Errorf("action %q not found: no integration named %q", name, integrationName)
		}
		return findIntegrationComponent(integration, componentName)
	}

	response, _, err := ctx.API.ActionAPI.ActionsDescribeAction(ctx.Context, name).Execute()
	if err != nil {
		return openapi_client.SuperplaneActionsAction{}, err
	}

	return response.GetAction(), nil
}

func findIntegrationComponent(integration openapi_client.IntegrationsIntegrationDefinition, name string) (openapi_client.SuperplaneActionsAction, error) {
	for _, action := range integration.GetActions() {
		actionName := action.GetName()
		if actionName == name || actionName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return action, nil
		}
	}

	return openapi_client.SuperplaneActionsAction{}, fmt.Errorf("action %q not found in integration %q", name, integration.GetName())
}
