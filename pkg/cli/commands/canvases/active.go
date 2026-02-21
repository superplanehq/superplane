package canvases

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ActiveCommand struct{}

func (c *ActiveCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) == 1 {
		return c.setActiveByID(ctx, ctx.Args[0])
	}

	if !ctx.Renderer.IsText() {
		return fmt.Errorf("interactive canvas selection requires text output")
	}

	return c.setActiveInteractively(ctx)
}

func (c *ActiveCommand) setActiveByID(ctx core.CommandContext, canvasID string) error {
	canvasID = strings.TrimSpace(canvasID)
	if canvasID == "" {
		return fmt.Errorf("canvas id is required")
	}

	_, _, err := ctx.API.CanvasAPI.
		CanvasesDescribeCanvas(ctx.Context, canvasID).
		Execute()

	if err != nil {
		return err
	}

	return ctx.Config.SetActiveCanvas(canvasID)
}

func (c *ActiveCommand) setActiveInteractively(ctx core.CommandContext) error {
	response, _, err := ctx.API.CanvasAPI.
		CanvasesListCanvases(ctx.Context).
		Execute()

	if err != nil {
		return err
	}

	canvases := response.GetCanvases()
	if len(canvases) == 0 {
		return fmt.Errorf("no canvases found")
	}

	err = ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for i, canvas := range canvases {
			prefix := " "
			if *canvas.Metadata.Id == ctx.Config.GetActiveCanvas() {
				prefix = "*"
			}
			_, _ = fmt.Fprintf(stdout, "%s %d. %s (%s)\n", prefix, i+1, *canvas.Metadata.Name, *canvas.Metadata.Id)
		}
		_, _ = fmt.Fprint(stdout, "Select a canvas number: ")
		return nil
	})

	if err != nil {
		return err
	}

	reader := bufio.NewReader(ctx.Cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read selected canvas: %w", err)
	}

	selectedIndex, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return fmt.Errorf("invalid canvas selection %q", strings.TrimSpace(input))
	}

	if selectedIndex < 1 || selectedIndex > len(canvases) {
		return fmt.Errorf("canvas selection must be between 1 and %d", len(canvases))
	}

	selected := canvases[selectedIndex-1]
	return ctx.Config.SetActiveCanvas(*selected.Metadata.Id)
}
