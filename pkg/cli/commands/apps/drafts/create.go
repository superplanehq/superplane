package drafts

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	name *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	displayName := ""
	if c.name != nil {
		displayName = strings.TrimSpace(*c.name)
	}

	body := openapi_client.CanvasesCreateCanvasVersionBody{}
	if displayName != "" {
		body.SetDisplayName(displayName)
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesCreateCanvasVersion(ctx.Context, appID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}
	if response.Version == nil || response.Version.Metadata == nil {
		return fmt.Errorf("draft version was not returned by the API")
	}

	version := *response.Version
	versionID := strings.TrimSpace(version.Metadata.GetId())
	if versionID == "" {
		return fmt.Errorf("draft version id was not returned by the API")
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	nameOutput := displayName
	if nameOutput == "" {
		if version.Metadata.HasDisplayName() && strings.TrimSpace(version.Metadata.GetDisplayName()) != "" {
			nameOutput = strings.TrimSpace(version.Metadata.GetDisplayName())
		} else {
			nameOutput = "(none)"
		}
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Draft created for app %s\n", appID)
		_, _ = fmt.Fprintf(stdout, "Draft ID: %s\n", versionID)
		_, err := fmt.Fprintf(stdout, "Name:     %s\n", nameOutput)
		return err
	})
}
