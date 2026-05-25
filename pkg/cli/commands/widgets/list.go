package widgets

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type listCommand struct {
	canvasID *string
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	canvas, err := fetchCanvas(ctx, canvasID)
	if err != nil {
		return err
	}

	spec := canvas.GetSpec()
	widgets := []map[string]any{}
	for _, node := range spec.GetNodes() {
		if !canvasNodeIsWidget(node) {
			continue
		}
		row := map[string]any{
			"id":        node.GetId(),
			"name":      node.GetName(),
			"component": node.GetComponent(),
		}
		if node.Position != nil {
			row["position"] = map[string]int32{
				"x": node.Position.GetX(),
				"y": node.Position.GetY(),
			}
		}
		widgets = append(widgets, row)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(widgets)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(widgets) == 0 {
			_, err := fmt.Fprintln(stdout, "No widget nodes on this canvas.")
			return err
		}
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCOMPONENT\tPOSITION")
		for _, w := range widgets {
			pos := "-"
			if posMap, ok := w["position"].(map[string]int32); ok {
				pos = fmt.Sprintf("%d,%d", posMap["x"], posMap["y"])
			}
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", w["id"], w["name"], w["component"], pos)
		}
		return writer.Flush()
	})
}

func valueOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
