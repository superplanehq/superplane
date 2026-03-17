package extensions

import (
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type PublishVersionCommand struct {
	ExtensionID string
	Version     string
}

func (c *PublishVersionCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.ExtensionAPI.ExtensionsPublishVersion(ctx.Context, c.ExtensionID, c.Version).Execute()
	if err != nil {
		return err
	}

	version := response.GetVersion()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return nil
}
