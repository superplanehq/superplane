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
	file    *string
	draftID *string
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
	draftID := ""
	if c.draftID != nil {
		draftID = strings.TrimSpace(*c.draftID)
	}
	draftOnly := draftID != ""

	yamlBytes, source, err := resolveYAMLSource(ctx.Cmd.InOrStdin(), flagValue, positional)
	if err != nil {
		return err
	}

	if _, err := ParseConsoleYAML(yamlBytes); err != nil {
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

	var versionID string
	if draftOnly {
		versionID, err = common.ResolveDraftVersionID(ctx, canvasID, draftID)
	} else {
		versionID, err = common.EnsureCurrentUserDraftVersionID(ctx, canvasID)
	}
	if err != nil {
		return err
	}

	if err := common.StageCommitRepositorySpecFile(
		ctx,
		canvasID,
		versionID,
		common.ConsoleYAMLRepositoryPath,
		yamlBytes,
		nil,
	); err != nil {
		return err
	}

	updatedYAML, err := common.FetchRepositoryFile(ctx, canvasID, common.ConsoleYAMLRepositoryPath, versionID)
	if err != nil {
		return fmt.Errorf("console draft updated but failed to read console.yaml: %w", err)
	}

	updatedResource, err := ParseConsoleYAML(updatedYAML)
	if err != nil {
		return fmt.Errorf("invalid console yaml from server: %w", err)
	}

	// When change management is enabled, drafts are not visible from the
	// UI on their own; the user can only see/approve them via a change
	// request. Auto-create one so the operator sees the result of the
	// command in the UI without a follow-up call. Pass --draft-id to skip.
	var createdChangeRequestID string
	if changeManagementEnabled && !draftOnly {
		createdChangeRequestID, err = createChangeRequestForDraft(ctx, canvasID, versionID)
		if err != nil {
			return fmt.Errorf("console draft updated but failed to create change request: %w", err)
		}
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updatedResource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Console draft updated for app %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Draft version: %s\n", versionID)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(updatedResource.Spec.Panels))
		_, _ = fmt.Fprintf(stdout, "Layout items: %d\n", len(updatedResource.Spec.Layout))
		if createdChangeRequestID != "" {
			_, err := fmt.Fprintf(stdout, "Change request: %s (open)\n", createdChangeRequestID)
			return err
		}
		if changeManagementEnabled {
			_, err := fmt.Fprintln(stdout, "Run `superplane apps change-requests create` to open a change request for this draft.")
			return err
		}
		_, err := fmt.Fprintln(stdout, "Run `superplane apps canvas update` (without --draft-id) to publish a draft that includes this console.")
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
