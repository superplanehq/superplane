package drafts

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
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

	liveVersionID, err := common.EnsureLiveVersionID(ctx, appID)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]string{
			"appId":     appID,
			"versionId": liveVersionID,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "App %s edits are staged against the live canvas.\n", appID)
		_, err := fmt.Fprintf(stdout, "Live version: %s\n", liveVersionID)
		return err
	})
}
