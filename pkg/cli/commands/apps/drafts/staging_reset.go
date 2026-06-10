package drafts

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type stagingResetCommand struct {
	paths *[]string
}

func (c *stagingResetCommand) Execute(ctx core.CommandContext) error {
	draftID, appID, err := resolveStagingDraftTarget(ctx)
	if err != nil {
		return err
	}

	paths := []string{}
	if c.paths != nil {
		for _, path := range *c.paths {
			trimmed := strings.TrimSpace(path)
			if trimmed == "" {
				continue
			}
			paths = append(paths, trimmed)
		}
	}

	if err := common.DiscardCanvasStaging(ctx, appID, draftID, paths); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		payload := map[string]any{
			"draftId": draftID,
			"appId":   appID,
			"status":  "reset",
		}
		if len(paths) > 0 {
			payload["paths"] = paths
		}
		return ctx.Renderer.Render(payload)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(paths) == 0 {
			_, err := fmt.Fprintf(stdout, "All staging discarded for draft %s (app %s)\n", draftID, appID)
			return err
		}
		_, err := fmt.Fprintf(stdout, "Staging discarded for %s on draft %s (app %s)\n", strings.Join(paths, ", "), draftID, appID)
		return err
	})
}
