package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type setCommand struct {
	file      *string
	draftOnly *bool
}

func (c *setCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 2 {
		return fmt.Errorf("unexpected extra arguments; usage: superplane apps console set [app-name-or-id] [file]")
	}

	canvasArg := ""
	if len(ctx.Args) >= 1 {
		canvasArg = strings.TrimSpace(ctx.Args[0])
	}
	positional := ""
	if len(ctx.Args) == 2 {
		positional = strings.TrimSpace(ctx.Args[1])
	}

	flagValue := ""
	if c.file != nil {
		flagValue = strings.TrimSpace(*c.file)
	}
	draftOnly := c.draftOnly != nil && *c.draftOnly

	yamlBytes, source, err := resolveYAMLSource(ctx.Cmd.InOrStdin(), flagValue, positional)
	if err != nil {
		return err
	}

	resource, err := ParseConsoleYAML(yamlBytes)
	if err != nil {
		return fmt.Errorf("invalid console yaml in %s: %w", source, err)
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	changeManagementEnabled, err := common.ChangeManagementEnabled(ctx, canvasID)
	if err != nil {
		return err
	}

	versionID, err := common.EnsureCurrentUserDraftVersionID(ctx, canvasID)
	if err != nil {
		return err
	}

	body := openapi_client.CanvasesUpdateConsoleBody{}
	body.SetVersionId(versionID)
	panels, err := apiPanelsFromYAML(resource.Spec.Panels)
	if err != nil {
		return fmt.Errorf("invalid console yaml in %s: %w", source, err)
	}
	body.SetPanels(panels)
	body.SetLayout(apiLayoutFromYAML(resource.Spec.Layout))

	response, _, err := ctx.API.CanvasAPI.
		CanvasesUpdateConsole(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}
	if response.Console == nil {
		return fmt.Errorf("update succeeded but server did not return a console")
	}

	console := *response.Console

	// When change management is enabled, drafts are not visible from the
	// UI on their own; the user can only see/approve them via a change
	// request. Auto-create one so the operator sees the result of the
	// command in the UI without a follow-up call. Pass --draft to skip.
	var createdChangeRequestID string
	if changeManagementEnabled && !draftOnly {
		createdChangeRequestID, err = createChangeRequestForDraft(ctx, canvasID, versionID)
		if err != nil {
			return fmt.Errorf("console draft updated but failed to create change request: %w", err)
		}
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(consoleYAMLFromAPI(resource.Metadata.Name, console))
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Console draft updated for app %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Draft version: %s\n", strings.TrimSpace(console.GetVersionId()))
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(console.GetPanels()))
		_, _ = fmt.Fprintf(stdout, "Layout items: %d\n", len(console.GetLayout()))
		if createdChangeRequestID != "" {
			_, err := fmt.Fprintf(stdout, "Change request: %s (open)\n", createdChangeRequestID)
			return err
		}
		if changeManagementEnabled {
			_, err := fmt.Fprintln(stdout, "Run `superplane apps change-requests create` to open a change request for this draft.")
			return err
		}
		_, err := fmt.Fprintln(stdout, "Run `superplane apps canvas update` (without --draft) to publish a draft that includes this console.")
		return err
	})
}

// createChangeRequestForDraft opens a change request for the supplied
// draft version. It returns the change request id (or empty when the API
// does not echo one back).
func createChangeRequestForDraft(ctx core.CommandContext, canvasID string, versionID string) (string, error) {
	body := openapi_client.CanvasesCreateCanvasChangeRequestBody{}
	body.SetVersionId(versionID)

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesCreateCanvasChangeRequest(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return "", err
	}
	if response.ChangeRequest == nil || response.ChangeRequest.Metadata == nil {
		return "", nil
	}

	return strings.TrimSpace(response.ChangeRequest.Metadata.GetId()), nil
}
