package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/yaml"
)

type setCommand struct {
	file    *string
	message *string
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

	commitMessage, err := common.RequireCommitMessage(messageValue(c.message))
	if err != nil {
		return fmt.Errorf("%w; use \"superplane apps staging update\" and \"superplane apps staging commit\" to stage changes first", err)
	}

	yamlBytes, source, err := resolveYAMLSource(ctx.Cmd.InOrStdin(), flagValue, positional)
	if err != nil {
		return err
	}

	// Lenient parse — the server-side commit path applies a delta cap
	// check against the previously committed pages, so a migrated
	// (grandfathered) console with more than MaxConsolePanelsPerPage on
	// a single page can still be updated via the CLI as long as the
	// staged content does not push any page beyond both the cap and
	// its previously committed size. Structural errors (malformed
	// YAML, unknown fields, unsupported panel types) still surface.
	_, err = yaml.ConsoleFromYMLLenient(yamlBytes)
	if err != nil {
		return fmt.Errorf("invalid console yaml in %s: %w", source, err)
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	if err := common.StageRepositorySpecFile(
		ctx,
		canvasID,
		common.ConsoleYAMLRepositoryPath,
		yamlBytes,
	); err != nil {
		return err
	}

	commitResponse, err := common.CommitCanvasStaging(ctx, canvasID, commitMessage)
	if err != nil {
		return fmt.Errorf("console was staged but commit failed: %w", err)
	}

	version := commitResponse.GetVersion()
	if version.Metadata == nil {
		return fmt.Errorf("committed version metadata is missing")
	}
	versionID := strings.TrimSpace(version.Metadata.GetId())

	updatedYAML, err := common.FetchRepositoryFile(ctx, canvasID, common.ConsoleYAMLRepositoryPath, versionID)
	if err != nil {
		return fmt.Errorf("console updated but failed to read console.yaml: %w", err)
	}

	// Lenient again: the server may return a multi-page shape, and we
	// still want to display something useful for grandfathered consoles
	// even though a strict parse would reject them.
	updatedResource, err := yaml.ConsoleFromYMLLenient(updatedYAML)
	if err != nil {
		return fmt.Errorf("invalid console yaml from server: %w", err)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updatedResource)
	}

	// The legacy `spec.panels` / `spec.layout` fields are empty for
	// consoles parsed from the multi-page shape, so summing over
	// `Pages()` is the only accurate count regardless of how the YAML
	// was written.
	panelCount := 0
	layoutCount := 0
	for _, page := range updatedResource.Pages() {
		panelCount += len(page.Panels)
		layoutCount += len(page.Layout)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Console updated for app %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Version: %s\n", versionID)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", panelCount)
		_, err := fmt.Fprintf(stdout, "Layout items: %d\n", layoutCount)
		return err
	})
}

func messageValue(message *string) string {
	if message == nil {
		return ""
	}
	return *message
}
