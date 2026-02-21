package canvases

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	file *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}

	if filePath != "" {
		if len(ctx.Args) > 0 {
			return fmt.Errorf("cannot use <canvas-name> together with --file")
		}
		return c.createFromFile(ctx, filePath)
	}

	if len(ctx.Args) != 1 {
		return fmt.Errorf("either --file or <canvas-name> is required")
	}

	name := ctx.Args[0]
	resource := models.Canvas{
		APIVersion: core.APIVersion,
		Kind:       models.CanvasKind,
		Metadata:   &openapi_client.CanvasesCanvasMetadata{Name: &name},
		Spec:       models.EmptyCanvasSpec(),
	}

	canvas := models.CanvasFromCanvas(resource)
	request := openapi_client.CanvasesCreateCanvasRequest{}
	request.SetCanvas(canvas)

	_, _, err := ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(request).Execute()
	return err
}

func (c *createCommand) createFromFile(ctx core.CommandContext, path string) error {
	// #nosec
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read resource file: %w", err)
	}

	_, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return err
	}

	switch kind {
	case models.CanvasKind:
		resource, err := models.ParseCanvas(data)
		if err != nil {
			return err
		}

		canvas := models.CanvasFromCanvas(*resource)
		request := openapi_client.CanvasesCreateCanvasRequest{}
		request.SetCanvas(canvas)

		_, _, err = ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(request).Execute()
		return err
	default:
		return fmt.Errorf("unsupported resource kind %q", kind)
	}
}
