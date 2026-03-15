package extensions

import (
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type PublishVersionCommand struct {
	ExtensionID string
	VersionID   string
	Version     string
}

func (c *PublishVersionCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.ExtensionAPI.ExtensionsPublishVersion(ctx.Context, c.ExtensionID, c.VersionID).
		Body(openapi_client.ExtensionsPublishVersionBody{Version: &c.Version}).
		Execute()
	if err != nil {
		return err
	}

	version := response.GetVersion()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return nil
}
