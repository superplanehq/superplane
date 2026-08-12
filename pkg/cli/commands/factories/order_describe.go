package factories

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// maxWorkOrderEventsLimit is the maximum page size the ListWorkOrderEvents
// endpoint supports. There's no way to fetch every event in one call, so
// "describe" always asks for this many and notes truncation in its output
// when there are more.
const maxWorkOrderEventsLimit = 200

type orderDescribeCommand struct {
	factory *string
	orderID *string
}

func (c *orderDescribeCommand) Execute(ctx core.CommandContext) error {
	orderID := strings.TrimSpace(stringValue(c.orderID))
	if orderID == "" {
		return fmt.Errorf("--order is required")
	}

	factoryID, err := ResolveFactoryID(ctx, stringValue(c.factory))
	if err != nil {
		return err
	}

	describeResponse, _, err := ctx.API.FactoryAPI.
		FactoriesDescribeWorkOrder(ctx.Context, factoryID, orderID).
		Execute()
	if err != nil {
		return err
	}
	order := describeResponse.GetOrder()

	eventsResponse, _, err := ctx.API.FactoryAPI.
		FactoriesListWorkOrderEvents(ctx.Context, factoryID, orderID).
		Limit(maxWorkOrderEventsLimit).
		Execute()
	if err != nil {
		return err
	}

	// The API returns events newest-first; reverse them so the timeline
	// reads top to bottom chronologically, like a changelog.
	events := reverseWorkOrderEvents(eventsResponse.GetEvents())
	comments := filterCommentEvents(events)
	truncated := eventsResponse.GetHasNextPage()
	totalCount := eventsResponse.GetTotalCount()

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"order":           order,
			"comments":        comments,
			"events":          events,
			"eventsTruncated": truncated,
		})
	}

	lookup := resolveMemberEmailLookup(ctx, events)

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderOrderDescribeText(stdout, order, comments, events, truncated, totalCount, lookup)
	})
}

// resolveMemberEmailLookup fetches the organization's members so event actor
// user IDs can be rendered as emails instead of raw UUIDs. It's best-effort:
// when there are no events to render there's nothing to resolve, and when
// the members list can't be fetched (e.g. the caller lacks permission), it
// degrades gracefully to an empty lookup rather than failing the whole
// "describe" command — every actor then renders as "unknown user" instead.
func resolveMemberEmailLookup(ctx core.CommandContext, events []openapi_client.FactoriesWorkOrderEvent) memberEmailLookup {
	if len(events) == 0 {
		return memberEmailLookup{}
	}

	members, err := listOrganizationMembers(ctx)
	if err != nil {
		return memberEmailLookup{}
	}
	return newMemberEmailLookup(members)
}

func reverseWorkOrderEvents(events []openapi_client.FactoriesWorkOrderEvent) []openapi_client.FactoriesWorkOrderEvent {
	reversed := make([]openapi_client.FactoriesWorkOrderEvent, len(events))
	for i, event := range events {
		reversed[len(events)-1-i] = event
	}
	return reversed
}

func filterCommentEvents(events []openapi_client.FactoriesWorkOrderEvent) []openapi_client.FactoriesWorkOrderEvent {
	comments := make([]openapi_client.FactoriesWorkOrderEvent, 0, len(events))
	for _, event := range events {
		if event.GetType() == eventTypeOrderCommentAdded {
			comments = append(comments, event)
		}
	}
	return comments
}

func renderOrderDescribeText(
	stdout io.Writer,
	order openapi_client.FactoriesWorkOrder,
	comments []openapi_client.FactoriesWorkOrderEvent,
	events []openapi_client.FactoriesWorkOrderEvent,
	eventsTruncated bool,
	totalEventCount int64,
	lookup memberEmailLookup,
) error {
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	writeAlignedField(writer, "ID", order.GetId())
	writeAlignedField(writer, "Title", order.GetTitle())
	writeAlignedField(writer, "State", formatOrderState(order.GetState()))
	writeAlignedField(writer, "Result", formatOrderResult(order.GetResult()))
	writeAlignedField(writer, "Created", formatRelativeTime(order.GetCreatedAt()))
	writeAlignedField(writer, "Updated", formatRelativeTime(order.GetUpdatedAt()))
	writeAlignedField(writer, "Created By", formatWorkOrderCreator(order.GetCreatedBy()))
	if err := writer.Flush(); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Assignees:")
	assignees := order.GetAssignees()
	if len(assignees) == 0 {
		_, _ = fmt.Fprintln(stdout, "  (none)")
	} else {
		for _, assignee := range assignees {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", formatUserRef(assignee))
		}
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Description:")
	description := strings.TrimSpace(order.GetDescription())
	if description == "" {
		_, _ = fmt.Fprintln(stdout, "  (none)")
	} else {
		for _, line := range strings.Split(description, "\n") {
			_, _ = fmt.Fprintf(stdout, "  %s\n", line)
		}
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Comments:")
	if len(comments) == 0 {
		_, _ = fmt.Fprintln(stdout, "  No comments.")
	} else {
		for _, comment := range comments {
			author, body, ok := decodeCommentEvent(comment, lookup)
			if !ok {
				author, body = "unknown", ""
			}
			_, _ = fmt.Fprintf(stdout, "  %s %s: %s\n", formatRelativeTime(comment.GetTimestamp()), author, body)
		}
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Events:")
	if len(events) == 0 {
		_, _ = fmt.Fprintln(stdout, "  No events.")
	} else {
		eventsWriter := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		for _, event := range events {
			_, _ = fmt.Fprintf(
				eventsWriter,
				"  %s\t%s\t%s\n",
				event.GetType(),
				formatRelativeTime(event.GetTimestamp()),
				describeEvent(event, lookup),
			)
		}
		if err := eventsWriter.Flush(); err != nil {
			return err
		}
		if eventsTruncated {
			_, _ = fmt.Fprintf(stdout, "  (showing latest %d of %d events)\n", maxWorkOrderEventsLimit, totalEventCount)
		}
	}

	return nil
}

func writeAlignedField(w io.Writer, label, value string) {
	_, _ = fmt.Fprintf(w, "%s\t%s\n", label, value)
}
