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
	displayName *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	canvasArg := ""
	if len(ctx.Args) == 1 {
		canvasArg = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	body := openapi_client.CanvasesCreateDraftBranchBody{}
	if c.displayName != nil {
		if name := strings.TrimSpace(*c.displayName); name != "" {
			body.SetDisplayName(name)
		}
	}

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesCreateDraftBranch(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}
	if response.Branch == nil {
		return fmt.Errorf("draft branch was not returned by the API")
	}

	branch := *response.Branch
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(branch)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Draft branch: %s\n", branch.GetBranchName())
		if label := strings.TrimSpace(branch.GetDisplayName()); label != "" {
			_, _ = fmt.Fprintf(stdout, "Display name: %s\n", label)
		}
		if tip := strings.TrimSpace(branch.GetTipSha()); tip != "" {
			_, _ = fmt.Fprintf(stdout, "Tip SHA: %s\n", tip)
		}
		_, err := fmt.Fprintf(stdout, "App ID: %s\n", canvasID)
		return err
	})
}
