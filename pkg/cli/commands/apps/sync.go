package apps

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type syncCommand struct{}

func (c *syncCommand) Execute(ctx core.CommandContext) error {
	nameOrID := ctx.Args[0]

	appID, err := findAppID(ctx, nameOrID)
	if err != nil {
		return err
	}

	resp, _, err := ctx.API.AppAPI.AppsSyncApp(ctx.Context, appID).Body(map[string]any{}).Execute()
	if err != nil {
		return err
	}

	if resp.App == nil {
		return fmt.Errorf("sync completed but no app data was returned")
	}

	app := *resp.App
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(buildAppSummary(app))
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		syncState := app.GetSyncState()
		_, err := fmt.Fprintf(stdout, "App %q synced (status: %s)\n",
			nameOrID,
			syncState.GetStatus(),
		)
		return err
	})
}
