package apps

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type describeCommand struct{}

func (c *describeCommand) Execute(ctx core.CommandContext) error {
	appID, err := findAppID(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	response, _, err := ctx.API.AppAPI.AppsDescribeApp(ctx.Context, appID).Execute()
	if err != nil {
		return err
	}

	if response.App == nil {
		return fmt.Errorf("app %q not found", ctx.Args[0])
	}

	app := *response.App
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(buildAppSummary(app))
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := app.GetMetadata()
		syncState := app.GetSyncState()

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)

		_, _ = fmt.Fprintln(writer, "FIELD\tVALUE")
		_, _ = fmt.Fprintf(writer, "ID\t%s\n", metadata.GetId())
		_, _ = fmt.Fprintf(writer, "DISPLAY_NAME\t%s\n", metadata.GetDisplayName())
		_, _ = fmt.Fprintf(writer, "SLUG\t%s\n", metadata.GetSlug())

		if metadata.GetDescription() != "" {
			_, _ = fmt.Fprintf(writer, "DESCRIPTION\t%s\n", metadata.GetDescription())
		}

		if metadata.GetCanvasId() != "" {
			_, _ = fmt.Fprintf(writer, "CANVAS_ID\t%s\n", metadata.GetCanvasId())
		}

		if metadata.HasCreatedAt() {
			_, _ = fmt.Fprintf(writer, "CREATED_AT\t%s\n", metadata.GetCreatedAt().Format(time.RFC3339))
		}

		if metadata.HasUpdatedAt() {
			_, _ = fmt.Fprintf(writer, "UPDATED_AT\t%s\n", metadata.GetUpdatedAt().Format(time.RFC3339))
		}

		_, _ = fmt.Fprintln(writer, "")
		_, _ = fmt.Fprintln(writer, "SYNC STATE")
		_, _ = fmt.Fprintf(writer, "STATUS\t%s\n", syncState.GetStatus())

		if syncState.GetError() != "" {
			_, _ = fmt.Fprintf(writer, "ERROR\t%s\n", syncState.GetError())
		}

		if syncState.GetDefaultBranch() != "" {
			_, _ = fmt.Fprintf(writer, "DEFAULT_BRANCH\t%s\n", syncState.GetDefaultBranch())
		}

		if syncState.GetLiveCommitSha() != "" {
			_, _ = fmt.Fprintf(writer, "LIVE_COMMIT_SHA\t%s\n", syncState.GetLiveCommitSha())
		}

		if syncState.GetCodeStorageRemoteUrl() != "" {
			_, _ = fmt.Fprintf(writer, "CODE_STORAGE_REMOTE_URL\t%s\n", syncState.GetCodeStorageRemoteUrl())
		}

		if syncState.GetCodeStorageRepoId() != "" {
			_, _ = fmt.Fprintf(writer, "CODE_STORAGE_REPO_ID\t%s\n", syncState.GetCodeStorageRepoId())
		}

		if syncState.GetEditSessionBranch() != "" {
			_, _ = fmt.Fprintf(writer, "EDIT_SESSION_BRANCH\t%s\n", syncState.GetEditSessionBranch())
		}

		return writer.Flush()
	})
}

func buildAppSummary(app openapi_client.AppsApp) map[string]any {
	metadata := app.GetMetadata()
	syncState := app.GetSyncState()

	createdAt := ""
	if metadata.HasCreatedAt() {
		createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
	}

	updatedAt := ""
	if metadata.HasUpdatedAt() {
		updatedAt = metadata.GetUpdatedAt().Format(time.RFC3339)
	}

	return map[string]any{
		"id":          metadata.GetId(),
		"displayName": metadata.GetDisplayName(),
		"slug":        metadata.GetSlug(),
		"description": metadata.GetDescription(),
		"canvasId":    metadata.GetCanvasId(),
		"createdAt":   createdAt,
		"updatedAt":   updatedAt,
		"syncState": map[string]string{
			"status":               syncState.GetStatus(),
			"error":                syncState.GetError(),
			"defaultBranch":        syncState.GetDefaultBranch(),
			"liveCommitSha":        syncState.GetLiveCommitSha(),
			"codeStorageRemoteUrl": syncState.GetCodeStorageRemoteUrl(),
			"codeStorageRepoId":    syncState.GetCodeStorageRepoId(),
			"editSessionBranch":    syncState.GetEditSessionBranch(),
		},
	}
}
