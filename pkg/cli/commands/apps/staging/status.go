package staging

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type statusCommand struct{}

func (c *statusCommand) Execute(ctx core.CommandContext) error {
	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	summary, err := common.GetCanvasStaging(ctx, appID)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if !summary.GetHasStaging() {
			return nil
		}

		for _, path := range summary.GetStagedPaths() {
			if _, err := fmt.Fprintln(stdout, formatStagedPathLine(stdout, path)); err != nil {
				return err
			}
		}
		return nil
	})
}
