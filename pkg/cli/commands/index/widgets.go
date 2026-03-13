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

	cmd := &cobra.Command{
		Use:   "widgets",
		Short: "List or describe available widgets",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&name, "name", "", "widget name")
	core.Bind(cmd, &widgetsCommand{name: &name}, options)

	return cmd
}

type widgetsCommand struct {
	name *string
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
		return ctx.Renderer.Render(widgets)
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
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", widget.GetName())
		_, _ = fmt.Fprintf(stdout, "Label: %s\n", widget.GetLabel())
		_, err := fmt.Fprintf(stdout, "Description: %s\n", widget.GetDescription())
		return err
	})
}

func (c *widgetsCommand) findWidgetByName(ctx core.CommandContext, name string) (openapi_client.WidgetsWidget, error) {
	response, _, err := ctx.API.WidgetAPI.WidgetsDescribeWidget(ctx.Context, name).Execute()
	if err != nil {
		return openapi_client.WidgetsWidget{}, err
	}

	return response.GetWidget(), nil
}
