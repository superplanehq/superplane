package events

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListEventsCommand struct {
	CanvasID *string
	NodeID   *string
	Limit    *int64
	Before   *string
}

func (c *ListEventsCommand) Execute(ctx core.CommandContext) error {
	if c.NodeID != nil && *c.NodeID != "" {
		return c.listNodeEvents(ctx)
	}

	return c.listCanvasEvents(ctx)
}

func (c *ListEventsCommand) listNodeEvents(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasNodeAPI.
		CanvasesListNodeEvents(ctx.Context, canvasID, *c.NodeID)

	if c.Limit != nil && *c.Limit > 0 {
		request = request.Limit(*c.Limit)
	}

	if c.Before != nil && *c.Before != "" {
		beforeTime, err := time.Parse(time.RFC3339, *c.Before)
		if err != nil {
			return fmt.Errorf("invalid --before value %q: expected RFC3339 timestamp", *c.Before)
		}
		request = request.Before(beforeTime)
	}

	response, _, err := request.Execute()

	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tCHANNEL\tCREATED_AT")
		for _, event := range response.GetEvents() {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\n",
				event.GetId(),
				event.GetChannel(),
				event.GetCreatedAt().Format(time.RFC3339),
			)
		}

		return writer.Flush()
	})
}

func (c *ListEventsCommand) listCanvasEvents(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasEventAPI.
		CanvasesListCanvasEvents(ctx.Context, canvasID)

	if c.Limit != nil && *c.Limit > 0 {
		request = request.Limit(*c.Limit)
	}

	if c.Before != nil && *c.Before != "" {
		beforeTime, err := time.Parse(time.RFC3339, *c.Before)
		if err != nil {
			return fmt.Errorf("invalid --before value %q: expected RFC3339 timestamp", *c.Before)
		}
		request = request.Before(beforeTime)
	}

	response, _, err := request.Execute()

	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNODE_ID\tCHANNEL\tEXECUTIONS\tCREATED_AT")
		for _, event := range response.GetEvents() {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%d\t%s\n",
				event.GetId(),
				event.GetNodeId(),
				event.GetChannel(),
				len(event.GetExecutions()),
				event.GetCreatedAt().Format(time.RFC3339),
			)
		}

		return writer.Flush()
	})
}
