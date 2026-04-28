package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func printNodeMessages(w io.Writer, nodes []openapi_client.SuperplaneComponentsNode) (bool, error) {
	hasErrors := false
	for _, node := range nodes {
		if msg := node.GetErrorMessage(); msg != "" {
			if _, err := fmt.Fprintf(w, "Node %q error: %s\n", node.GetId(), msg); err != nil {
				return false, err
			}
			hasErrors = true
		}
		if msg := node.GetWarningMessage(); msg != "" {
			if _, err := fmt.Fprintf(w, "Node %q warning: %s\n", node.GetId(), msg); err != nil {
				return false, err
			}
		}
	}
	return hasErrors, nil
}

type validateCommand struct {
	file *string
}

func (c *validateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}

	canvas, err := loadCanvasForValidateFromFile(filePath)
	if err != nil {
		return err
	}

	request := openapi_client.CanvasesValidateCanvasRequest{}
	request.SetCanvas(canvas)

	resp, httpResp, err := ctx.API.CanvasAPI.CanvasesValidateCanvas(ctx.Context).Body(request).Execute()
	if err != nil {
		return err
	}

	if httpResp != nil && (httpResp.StatusCode < 200 || httpResp.StatusCode >= 300) {
		return fmt.Errorf("unexpected response status: %s", httpResp.Status)
	}

	if resp == nil {
		return fmt.Errorf("validate canvas: server returned an empty response")
	}

	version := resp.GetVersion()
	if !ctx.Renderer.IsText() {
		if err := ctx.Renderer.Render(version); err != nil {
			return err
		}
		spec := version.GetSpec()
		for _, node := range spec.GetNodes() {
			if node.GetErrorMessage() != "" {
				return fmt.Errorf("canvas has node errors")
			}
		}
		return nil
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		spec := version.GetSpec()
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", len(spec.GetNodes()))
		_, _ = fmt.Fprintf(stdout, "Edges: %d\n", len(spec.GetEdges()))
		hasErrors, err := printNodeMessages(stdout, spec.GetNodes())
		if err != nil {
			return err
		}
		if hasErrors {
			return fmt.Errorf("canvas has node errors")
		}
		_, _ = fmt.Fprintf(stdout, "Canvas is valid\n")
		return nil
	})
}
