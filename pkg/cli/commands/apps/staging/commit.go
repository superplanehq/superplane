package staging

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type commitCommand struct {
	message *string
}

func (c *commitCommand) Execute(ctx core.CommandContext) error {
	message := ""
	if c.message != nil {
		message = *c.message
	}

	commitMessage, err := common.RequireCommitMessage(message)
	if err != nil {
		return err
	}

	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	commitResponse, err := common.CommitCanvasStaging(ctx, appID, commitMessage)
	if err != nil {
		return err
	}

	version := commitResponse.GetVersion()
	if version.Metadata == nil {
		return fmt.Errorf("committed version metadata is missing")
	}
	versionID := strings.TrimSpace(version.Metadata.GetId())
	if versionID == "" {
		return fmt.Errorf("committed version id is missing")
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Committed staged changes for app %s\n", appID)
		_, err := fmt.Fprintf(stdout, "Version: %s\n", versionID)
		return err
	})
}
