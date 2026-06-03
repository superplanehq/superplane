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
	canvasArg := ""
	if len(ctx.Args) == 1 {
		canvasArg = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesListDraftBranches(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return err
	}

	branches := response.GetBranches()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"canvasId": canvasID,
			"branches": branches,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(branches) == 0 {
			_, err := fmt.Fprintln(stdout, "No draft branches.")
			return err
		}

		w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "BRANCH\tDISPLAY\tTIP SHA\tSTATUS")
		for _, branch := range branches {
			status := strings.TrimSpace(branch.GetMaterializationStatus())
			if status == "" {
				status = "-"
			}
			_, _ = fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\n",
				branch.GetBranchName(),
				branch.GetDisplayName(),
				shortSHA(branch.GetTipSha()),
				status,
			)
		}
		return w.Flush()
	})
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}
