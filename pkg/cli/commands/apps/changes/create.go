package changes

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type CreateCommand struct {
	draftID     *string
	title       *string
	description *string
}

func (c *CreateCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("create accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, target)
	if err != nil {
		return err
	}

	versionID := ""
	if c.draftID != nil {
		versionID = strings.TrimSpace(*c.draftID)
	}

	if versionID == "" {
		changeManagementEnabled, err := common.ChangeManagementEnabled(ctx, canvasID)
		if err != nil {
			return err
		}
		if !changeManagementEnabled {
			return fmt.Errorf("change management is disabled for this canvas; enable it in canvas settings to use change requests")
		}

		versionID, err = common.FindCurrentUserDraftVersionID(ctx, canvasID)
		if err != nil {
			return err
		}
		if versionID == "" {
			return fmt.Errorf("no draft version found; run `superplane apps drafts create` then `superplane apps canvas update --draft-id <id> -f <file>` first")
		}
	}

	body := openapi_client.CanvasesCreateCanvasChangeRequestBody{}
	body.SetVersionId(versionID)

	if c.title != nil {
		trimmedTitle := strings.TrimSpace(*c.title)
		if trimmedTitle != "" {
			body.SetTitle(trimmedTitle)
		}
	}
	if c.description != nil {
		trimmedDescription := strings.TrimSpace(*c.description)
		if trimmedDescription != "" {
			body.SetDescription(trimmedDescription)
		}
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesCreateCanvasChangeRequest(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if response.ChangeRequest == nil {
		return nil
	}

	changeRequest := *response.ChangeRequest
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(changeRequest)
	}

	return renderCanvasChangeRequestSummaryText(ctx, "created", changeRequest)
}
