package extensions

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type UpdateVersionCommand struct {
	ExtensionID string
	Version     string
	EntryPoint  string
	Watch       bool
}

func (c *UpdateVersionCommand) Execute(ctx core.CommandContext) error {
	if c.Watch && !ctx.Renderer.IsText() {
		return fmt.Errorf("--watch only supports text output")
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}

	entryPoint, err := resolveEntryPoint(projectDir, c.EntryPoint)
	if err != nil {
		return err
	}

	bundle, digest, err := buildExtensionVersionUpload(ctx.Context, projectDir, entryPoint)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.ExtensionAPI.ExtensionsUpdateVersion(ctx.Context, c.ExtensionID, c.Version).
		Body(openapi_client.ExtensionsUpdateVersionBody{
			Bundle: &bundle,
			Digest: &digest,
		}).
		Execute()
	if err != nil {
		return err
	}

	version := response.GetVersion()
	if c.Watch {
		_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "Updated draft version %s. Watching for changes...\n", c.Version)
		return watchAndUpdateVersion(ctx, c.ExtensionID, projectDir, entryPoint, c.Version)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return nil
}
