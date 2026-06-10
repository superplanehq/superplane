package drafts

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type stagingCommitCommand struct{}

func (c *stagingCommitCommand) Execute(ctx core.CommandContext) error {
	draftID, appID, err := resolveStagingDraftTarget(ctx)
	if err != nil {
		return err
	}

	if err := common.CommitCanvasStaging(ctx, appID, draftID); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]string{
			"draftId": draftID,
			"appId":   appID,
			"status":  "committed",
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Staging committed for draft %s (app %s)\n", draftID, appID)
		_, err := fmt.Fprintln(stdout, "Run `superplane apps canvas update` without --stage-only to publish, or use the UI Publish action.")
		return err
	})
}
