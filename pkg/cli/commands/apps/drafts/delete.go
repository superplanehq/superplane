package drafts

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type deleteCommand struct{}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	if err := common.DiscardCanvasStaging(ctx, appID); err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Staged changes discarded for app %s.\n", appID)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]string{
		"appId":   appID,
		"deleted": "true",
	})
}
