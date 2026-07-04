package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
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

	var versionID string
	if draftOnly {
		versionID, err = common.ResolveDraftVersionID(ctx, canvasID, draftID)
	} else {
		versionID, err = common.EnsureCurrentUserDraftVersionID(ctx, canvasID)
	}
	if err != nil {
		return err
	}

	if err := common.CommitRepositorySpecFile(
		ctx,
		canvasID,
		versionID,
		common.ConsoleYAMLRepositoryPath,
		yamlBytes,
		"Update console.yaml",
		nil,
		false,
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

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updatedResource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Console draft updated for app %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Draft version: %s\n", versionID)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(updatedResource.Spec.Panels))
		_, _ = fmt.Fprintf(stdout, "Layout items: %d\n", len(updatedResource.Spec.Layout))
		_, err := fmt.Fprintln(stdout, "Run `superplane apps canvas update` (without --draft-id) to publish a draft that includes this console.")
		return err
	})
}
