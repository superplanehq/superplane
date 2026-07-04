package drafts

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type listCommand struct {
	all *bool
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	versions, err := common.ListDraftVersions(ctx, appID)
	if err != nil {
		return err
	}

	showAll := c.all != nil && *c.all
	currentUserID := ""
	if !showAll {
		currentUserID, err = currentUserIDFromContext(ctx)
		if err != nil {
			return err
		}
		versions = filterOwnedDrafts(versions, currentUserID)
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return updatedAt(versions[i]).After(updatedAt(versions[j]))
	})

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(versions)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(versions) == 0 {
			_, err := fmt.Fprintf(stdout, "No drafts found for app %s.\n", appID)
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "DRAFT ID\tNAME\tOWNER\tUPDATED")

		for _, version := range versions {
			metadata := version.GetMetadata()
			name := draftDisplayName(metadata)
			owner := formatDraftOwner(metadata, currentUserID)
			updated := formatRelativeTime(updatedAt(version))

			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\n",
				metadata.GetId(),
				name,
				owner,
				updated,
			)
		}

		return writer.Flush()
	})
}

func currentUserIDFromContext(ctx core.CommandContext) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	currentUserID := strings.TrimSpace(me.User.GetId())
	if currentUserID == "" {
		return "", fmt.Errorf("current user id not found")
	}

	return currentUserID, nil
}

func filterOwnedDrafts(versions []openapi_client.CanvasesCanvasVersion, userID string) []openapi_client.CanvasesCanvasVersion {
	var filtered []openapi_client.CanvasesCanvasVersion
	for _, version := range versions {
		ownerID := ""
		if version.Metadata != nil && version.Metadata.Owner != nil {
			ownerID = strings.TrimSpace(version.Metadata.Owner.GetId())
		}
		if ownerID != "" && strings.EqualFold(ownerID, userID) {
			filtered = append(filtered, version)
		}
	}
	return filtered
}

func draftDisplayName(metadata openapi_client.CanvasesCanvasVersionMetadata) string {
	if metadata.HasDisplayName() {
		if name := strings.TrimSpace(metadata.GetDisplayName()); name != "" {
			return name
		}
	}
	return "(none)"
}

func formatDraftOwner(metadata openapi_client.CanvasesCanvasVersionMetadata, currentUserID string) string {
	if metadata.Owner == nil {
		return "-"
	}

	ownerID := strings.TrimSpace(metadata.Owner.GetId())
	if currentUserID != "" && strings.EqualFold(ownerID, currentUserID) {
		return "you"
	}

	if email := strings.TrimSpace(metadata.Owner.GetName()); email != "" {
		return email
	}

	if ownerID != "" {
		return ownerID
	}

	return "-"
}

func updatedAt(version openapi_client.CanvasesCanvasVersion) time.Time {
	metadata := version.GetMetadata()
	if metadata.HasUpdatedAt() {
		return metadata.GetUpdatedAt()
	}
	return time.Time{}
}

func formatRelativeTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}

	elapsed := time.Since(value)
	if elapsed < 0 {
		elapsed = 0
	}

	switch {
	case elapsed < time.Minute:
		seconds := int(elapsed.Seconds())
		if seconds <= 1 {
			return "1s ago"
		}
		return fmt.Sprintf("%ds ago", seconds)
	case elapsed < time.Hour:
		minutes := int(elapsed.Minutes())
		if minutes <= 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	case elapsed < 24*time.Hour:
		hours := int(elapsed.Hours())
		if hours <= 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(elapsed.Hours() / 24)
		if days <= 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}
