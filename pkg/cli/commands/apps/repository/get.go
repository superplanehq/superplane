package repository

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type GetCommand struct{}

func (c *GetCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("get accepts at most one positional argument")
	}

	canvasTarget := ""
	if len(ctx.Args) == 1 {
		canvasTarget = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesGetCanvasRepository(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return err
	}

	repository, ok := response.GetRepositoryOk()
	if !ok || repository == nil {
		return fmt.Errorf("repository not found for app %q", canvasID)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(repository)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderRepositoryText(stdout, *repository)
	})
}

func renderRepositoryText(stdout io.Writer, repository openapi_client.CanvasesCanvasRepository) error {
	if metadata, ok := repository.GetMetadataOk(); ok && metadata != nil {
		if id := strings.TrimSpace(metadata.GetCanvasId()); id != "" {
			_, _ = fmt.Fprintf(stdout, "Canvas ID: %s\n", id)
		}
		if repoID := strings.TrimSpace(metadata.GetRepoId()); repoID != "" {
			_, _ = fmt.Fprintf(stdout, "Repository ID: %s\n", repoID)
		}
		if provider := strings.TrimSpace(metadata.GetProvider()); provider != "" {
			_, _ = fmt.Fprintf(stdout, "Provider: %s\n", provider)
		}
		if url := strings.TrimSpace(metadata.GetUrl()); url != "" {
			_, _ = fmt.Fprintf(stdout, "Clone URL: %s\n", url)
		}
		if branch := strings.TrimSpace(metadata.GetDefaultBranch()); branch != "" {
			_, _ = fmt.Fprintf(stdout, "Default branch: %s\n", branch)
		}
	}

	if status, ok := repository.GetStatusOk(); ok && status != nil {
		state := status.GetState()
		if state != openapi_client.CANVASESCANVASREPOSITORYSTATE_STATE_UNSPECIFIED {
			_, _ = fmt.Fprintf(stdout, "State: %s\n", state)
		}
		if headSHA := strings.TrimSpace(status.GetHeadSha()); headSHA != "" {
			_, _ = fmt.Fprintf(stdout, "Head SHA: %s\n", headSHA)
		}
		if status.Error != nil {
			if errMessage := strings.TrimSpace(status.GetError()); errMessage != "" {
				_, _ = fmt.Fprintf(stdout, "Error: %s\n", errMessage)
			}
		}
	}

	return nil
}
