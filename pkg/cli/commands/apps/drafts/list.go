package drafts

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type listCommand struct{}

func (c *listCommand) Execute(ctx core.CommandContext) error {
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
			_, err := fmt.Fprintf(stdout, "No staged changes for app %s.\n", appID)
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintf(writer, "STAGED PATHS\tSTALE\tBASE VERSION\n")
		baseVersion := strings.TrimSpace(summary.GetBaseVersionId())
		if baseVersion == "" {
			baseVersion = "(unknown)"
		}
		_, _ = fmt.Fprintf(
			writer,
			"%s\t%t\t%s\n",
			strings.Join(summary.GetStagedPaths(), ", "),
			summary.GetStale(),
			baseVersion,
		)
		return writer.Flush()
	})
}
