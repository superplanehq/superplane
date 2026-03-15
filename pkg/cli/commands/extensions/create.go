package extensions

import (
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type CreateCommand struct {
	Name        string
	Description string
}

func (c *CreateCommand) Execute(ctx core.CommandContext) error {
	request := openapi_client.ExtensionsCreateExtensionRequest{
		Name: &c.Name,
	}

	if c.Description != "" {
		request.SetDescription(c.Description)
	}

	response, _, err := ctx.API.ExtensionAPI.ExtensionsCreateExtension(ctx.Context).Body(request).Execute()
	if err != nil {
		return err
	}

	extension := response.GetExtension()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(extension)
	}

	return nil
}
