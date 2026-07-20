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

	_, err = yaml.ConsoleFromYML(yamlBytes)
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

	updatedResource, err := yaml.ConsoleFromYML(updatedYAML)
	if err != nil {
		return fmt.Errorf("invalid console yaml from server: %w", err)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updatedResource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Console updated for app %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Version: %s\n", versionID)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(updatedResource.Spec.Panels))
		_, err := fmt.Fprintf(stdout, "Layout items: %d\n", len(updatedResource.Spec.Layout))
		return err
	})
}

func messageValue(message *string) string {
	if message == nil {
		return ""
	}
	return *message
}
