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
	branchName := strings.TrimSpace(ctx.Args[0])
	if branchName == "" {
		return fmt.Errorf("branch name is required")
	}

	canvasArg := ""
	if len(ctx.Args) == 2 {
		canvasArg = strings.TrimSpace(ctx.Args[1])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	_, _, err = ctx.API.CanvasRepositoryAPI.
		CanvasesDeleteDraftBranch(ctx.Context, canvasID).
		BranchName(branchName).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"canvasId":   canvasID,
			"branchName": branchName,
			"deleted":    true,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Deleted draft branch %q for app %s\n", branchName, canvasID)
		return err
	})
}
