package changes

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListCommand struct {
	statusFilter *string
	onlyMine     *bool
	query        *string
	limit        *int64
	before       *string
}

func (c *ListCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("list accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, target)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasChangeRequestAPI.
		CanvasesListCanvasChangeRequests(ctx.Context, canvasID)

	if c.statusFilter != nil {
		statusFilter := strings.TrimSpace(*c.statusFilter)
		if statusFilter != "" {
			request = request.StatusFilter(statusFilter)
		}
	}
	if c.onlyMine != nil {
		request = request.OnlyMine(*c.onlyMine)
	}
	if c.query != nil {
		query := strings.TrimSpace(*c.query)
		if query != "" {
			request = request.Query(query)
		}
	}
	if c.limit != nil && *c.limit > 0 {
		request = request.Limit(*c.limit)
	}
	if c.before != nil {
		beforeRaw := strings.TrimSpace(*c.before)
		if beforeRaw != "" {
			beforeTime, parseErr := time.Parse(time.RFC3339, beforeRaw)
			if parseErr != nil {
				return fmt.Errorf("invalid --before value %q: expected RFC3339 timestamp", beforeRaw)
			}
			request = request.Before(beforeTime)
		}
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	changeRequests := response.GetChangeRequests()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(changeRequests)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(changeRequests) == 0 {
			_, err := fmt.Fprintln(stdout, "No change requests found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tSTATUS\tCONFLICTED\tCHANGED_NODES\tCONFLICTING_NODES\tTITLE\tUPDATED_AT")

		for _, changeRequest := range changeRequests {
			metadata := changeRequest.GetMetadata()
			diff := changeRequest.GetDiff()

			title := "-"
			if metadata.HasTitle() && strings.TrimSpace(metadata.GetTitle()) != "" {
				title = metadata.GetTitle()
			}

			updatedAt := ""
			if metadata.HasUpdatedAt() {
				updatedAt = metadata.GetUpdatedAt().Format(time.RFC3339)
			}

			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%t\t%d\t%d\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetStatus(),
				metadata.GetIsConflicted(),
				len(diff.GetChangedNodeIds()),
				len(diff.GetConflictingNodeIds()),
				title,
				updatedAt,
			)
		}

		return writer.Flush()
	})
}
