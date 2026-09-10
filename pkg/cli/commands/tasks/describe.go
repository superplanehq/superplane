package tasks

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const maxTaskEventsLimit = 200

type taskDescribeCommand struct {
	workspace *string
	taskID    *string
}

func (c *taskDescribeCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := strings.TrimSpace(stringValue(c.taskID))
	if rawTaskID == "" {
		return fmt.Errorf("--task is required")
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(ctx, workspaceID, rawTaskID)
	if err != nil {
		return err
	}

	describeResponse, _, err := ctx.API.FactoryAPI.
		FactoriesDescribeWorkOrder(ctx.Context, workspaceID, taskID).
		Execute()
	if err != nil {
		return err
	}
	task := describeResponse.GetOrder()

	eventsResponse, _, err := ctx.API.FactoryAPI.
		FactoriesListWorkOrderEvents(ctx.Context, workspaceID, taskID).
		Limit(maxTaskEventsLimit).
		Execute()
	if err != nil {
		return err
	}

	events := reverseWorkOrderEvents(eventsResponse.GetEvents())
	comments := filterCommentEvents(events)
	truncated := eventsResponse.GetHasNextPage()
	totalCount := eventsResponse.GetTotalCount()

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"task":            task,
			"comments":        comments,
			"events":          events,
			"eventsTruncated": truncated,
		})
	}

	lookup := resolveMemberEmailLookup(ctx, events)

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderTaskDescribeText(stdout, task, comments, events, truncated, totalCount, lookup)
	})
}

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
		if event.GetType() == eventTypeTaskCommentAdded {
			comments = append(comments, event)
		}
	}
	return comments
}

func renderTaskDescribeText(
	stdout io.Writer,
	task openapi_client.FactoriesWorkOrder,
	comments []openapi_client.FactoriesWorkOrderEvent,
	events []openapi_client.FactoriesWorkOrderEvent,
	eventsTruncated bool,
	totalEventCount int64,
	lookup memberEmailLookup,
) error {
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	writeAlignedField(writer, "ID", task.GetId())
	writeAlignedField(writer, "Title", task.GetTitle())
	writeAlignedField(writer, "State", formatTaskState(task.GetState()))
	writeAlignedField(writer, "Result", formatTaskResult(task.GetResult()))
	writeAlignedField(writer, "Created", formatRelativeTime(task.GetCreatedAt()))
	writeAlignedField(writer, "Updated", formatRelativeTime(task.GetUpdatedAt()))
	writeAlignedField(writer, "Created By", formatTaskCreator(task.GetCreatedBy()))
	if err := writer.Flush(); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Assignees:")
	assignees := task.GetAssignees()
	if len(assignees) == 0 {
		_, _ = fmt.Fprintln(stdout, "  (none)")
	} else {
		for _, assignee := range assignees {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", formatUserRef(assignee))
		}
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Description:")
	description := strings.TrimSpace(task.GetDescription())
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
			_, _ = fmt.Fprintf(stdout, "  (showing latest %d of %d events)\n", maxTaskEventsLimit, totalEventCount)
		}
	}

	return nil
}

func writeAlignedField(w io.Writer, label, value string) {
	_, _ = fmt.Fprintf(w, "%s\t%s\n", label, value)
}
