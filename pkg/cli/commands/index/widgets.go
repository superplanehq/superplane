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

func newWidgetsCommand(options core.BindOptions) *cobra.Command {
	var name string
	var full bool

	cmd := &cobra.Command{
		Use:   "widgets",
		Short: "List or describe available widgets",
		Long: `List or describe available widgets.

Use -o json or -o yaml with --name to inspect configuration fields,
defaults, and field-level constraints. Pass --full when listing widgets
to include configuration in the json/yaml payload.`,
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringVar(&name, "name", "", "widget name")
	cmd.Flags().BoolVar(&full, "full", false, "show full output including all fields")
	core.Bind(cmd, &widgetsCommand{name: &name, full: &full}, options)

	return cmd
}

type widgetsCommand struct {
	name *string
	full *bool
}

func (c *widgetsCommand) Execute(ctx core.CommandContext) error {
	name := strings.TrimSpace(*c.name)
	if name != "" {
		return c.getWidgetByName(ctx, name)
	}

	widgets, err := c.listWidgets(ctx)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		if c.full != nil && *c.full {
			return ctx.Renderer.Render(widgets)
		}

		summary := make([]map[string]string, len(widgets))
		for i, widget := range widgets {
			summary[i] = map[string]string{
				"name":        widget.GetName(),
				"label":       widget.GetLabel(),
				"description": widget.GetDescription(),
			}
		}
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, widget := range widgets {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", widget.GetName(), widget.GetLabel(), widget.GetDescription())
		}
		return writer.Flush()
	})
}

func (c *widgetsCommand) listWidgets(ctx core.CommandContext) ([]openapi_client.WidgetsWidget, error) {
	response, _, err := ctx.API.WidgetAPI.WidgetsListWidgets(ctx.Context).Execute()
	if err != nil {
		return nil, err
	}

	return response.GetWidgets(), nil
}

func (c *widgetsCommand) getWidgetByName(ctx core.CommandContext, name string) error {
	widget, err := c.findWidgetByName(ctx, name)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(widget)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderWidgetText(stdout, widget)
	})
}

func (c *widgetsCommand) findWidgetByName(ctx core.CommandContext, name string) (openapi_client.WidgetsWidget, error) {
	response, _, err := ctx.API.WidgetAPI.WidgetsDescribeWidget(ctx.Context, name).Execute()
	if err != nil {
		return openapi_client.WidgetsWidget{}, err
	}

	return response.GetWidget(), nil
}

// renderWidgetText prints a widget's metadata and its configuration fields
// in the same tabular layout used by other index entries (actions,
// triggers). Color and icon are surfaced because canvas widget instances
// inherit them and users typically want to see what they're getting.
func renderWidgetText(stdout io.Writer, widget openapi_client.WidgetsWidget) error {
	if _, err := fmt.Fprintf(stdout, "Name: %s\n", widget.GetName()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Label: %s\n", widget.GetLabel()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Description: %s\n", widget.GetDescription()); err != nil {
		return err
	}
	if icon := widget.GetIcon(); icon != "" {
		if _, err := fmt.Fprintf(stdout, "Icon: %s\n", icon); err != nil {
			return err
		}
	}
	if color := widget.GetColor(); color != "" {
		if _, err := fmt.Fprintf(stdout, "Color: %s\n", color); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}

	return renderConfigurationText(stdout, widget.GetConfiguration())
}
