package console

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/canvas/materialize"
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

	branch, err := common.EnsureCurrentUserDraftBranch(ctx, canvasID)
	if err != nil {
		return err
	}
	branchName := strings.TrimSpace(branch.GetBranchName())
	expectedHead := strings.TrimSpace(branch.GetTipSha())

	operations := []openapi_client.CanvasesCanvasRepositoryFileOperation{}

	consoleOp := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
	consoleOp.SetPath(materialize.ConsoleFileName)
	consoleOp.SetContent(base64.StdEncoding.EncodeToString(yamlBytes))
	operations = append(operations, *consoleOp)

	canvasYAML, err := common.FetchRepositoryFile(ctx, canvasID, materialize.CanvasFileName, branchName)
	if err == nil && len(canvasYAML) > 0 {
		canvasOp := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
		canvasOp.SetPath(materialize.CanvasFileName)
		canvasOp.SetContent(base64.StdEncoding.EncodeToString(canvasYAML))
		operations = append(operations, *canvasOp)
	}

	commitSHA, err := common.CommitRepositoryFiles(
		ctx,
		canvasID,
		branchName,
		expectedHead,
		"Update console.yaml",
		operations,
	)
	if err != nil {
		return err
	}

	var createdChangeRequestID string
	if changeManagementEnabled && !draftOnly {
		createdChangeRequestID, err = createChangeRequestForDraft(ctx, canvasID, commitSHA)
		if err != nil {
			return fmt.Errorf("console committed but failed to create change request: %w", err)
		}
	}

	if !ctx.Renderer.IsText() {
		dashboard := openapi_client.NewCanvasesCanvasDashboard()
		dashboard.SetVersionId(commitSHA)
		dashboard.SetPanels(apiPanelsFromYAML(resource.Spec.Panels))
		dashboard.SetLayout(apiLayoutFromYAML(resource.Spec.Layout))
		return ctx.Renderer.Render(consoleYAMLFromAPI(resource.Metadata.Name, *dashboard))
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Console committed for app %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Branch: %s\n", branchName)
		_, _ = fmt.Fprintf(stdout, "Commit SHA: %s\n", commitSHA)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(resource.Spec.Panels))
		_, _ = fmt.Fprintf(stdout, "Layout items: %d\n", len(resource.Spec.Layout))
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
