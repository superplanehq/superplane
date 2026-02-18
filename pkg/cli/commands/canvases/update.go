package canvases

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file *string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}
	if len(ctx.Args) > 0 {
		return fmt.Errorf("update does not accept positional arguments")
	}

	// #nosec
	data, err := os.ReadFile(filePath)
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
		if resource.Metadata == nil || resource.Metadata.Id == nil || resource.Metadata.GetId() == "" {
			return fmt.Errorf("canvas metadata.id is required for update")
		}

		canvas := models.CanvasFromCanvas(*resource)
		body := openapi_client.CanvasesUpdateCanvasBody{}
		body.SetCanvas(canvas)

		_, _, err = ctx.API.CanvasAPI.
			CanvasesUpdateCanvas(ctx.Context, resource.Metadata.GetId()).
			Body(body).
			Execute()
		return err
	default:
		return fmt.Errorf("unsupported resource kind %q for update", kind)
	}
}
