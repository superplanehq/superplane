package tasks

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/workspaces"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type taskListCommand struct {
	workspace  *string
	assignees  *[]string
	states     *[]string
	results    *[]string
	unassigned *bool
}

func (c *taskListCommand) Execute(ctx core.CommandContext) error {
	workspaceID, err := workspaces.ResolveWorkspaceID(ctx, stringValue(c.workspace))
	if err != nil {
		return err
	}

	request := ctx.API.FactoryAPI.FactoriesListWorkOrders(ctx.Context, workspaceID)

	if c.assignees != nil && len(*c.assignees) > 0 {
		assigneeIDs, err := resolveAssigneeIDs(ctx, *c.assignees)
		if err != nil {
			return err
		}
		if len(assigneeIDs) > 0 {
			request = request.AssigneeIds(assigneeIDs)
		}
	}

	var rawStates []string
	if c.states != nil {
		rawStates = *c.states
	}
	if resolved := resolveStateFilter(rawStates); resolved != nil {
		request = request.States(normalizeFilterValues(resolved, shortTaskStateTokens))
	}

	if c.results != nil && len(*c.results) > 0 {
		request = request.Results(normalizeFilterValues(*c.results, shortTaskResultTokens))
	}

	if c.unassigned != nil && *c.unassigned {
		request = request.Unassigned(true)
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	tasks := response.GetOrders()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(tasks)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(tasks) == 0 {
			_, err := fmt.Fprintln(stdout, "No tasks found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NUMBER\tTITLE\tSTATE\tRESULT\tASSIGNEES\tEXECUTIONS\tCREATED")
		for _, task := range tasks {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
				taskDisplayID(task),
				task.GetTitle(),
				formatTaskState(task.GetState()),
				formatTaskResult(task.GetResult()),
				formatAssigneeList(task.GetAssignees()),
				countStepExecutions(task.GetLineDispatches()),
				formatRelativeTime(task.GetCreatedAt()),
			)
		}
		return writer.Flush()
	})
}

func countStepExecutions(dispatches []openapi_client.FactoriesWorkOrderLineDispatch) int {
	total := 0
	for _, dispatch := range dispatches {
		total += len(dispatch.GetStepExecutions())
	}
	return total
}

var shortTaskStateTokens = map[string]string{
	"draft":  "STATE_DRAFT",
	"open":   "STATE_OPEN",
	"closed": "STATE_CLOSED",
}

const taskStateAllToken = "all"

var defaultTaskStates = []string{"open"}

func resolveStateFilter(states []string) []string {
	for _, state := range states {
		if strings.EqualFold(strings.TrimSpace(state), taskStateAllToken) {
			return nil
		}
	}

	if len(states) == 0 {
		return defaultTaskStates
	}

	return states
}

var shortTaskResultTokens = map[string]string{
	"completed": "RESULT_COMPLETED",
	"rejected":  "RESULT_REJECTED",
	"failed":    "RESULT_FAILED",
}

func normalizeFilterValues(values []string, shortNames map[string]string) []string {
	normalized := make([]string, len(values))
	for i, value := range values {
		trimmed := strings.TrimSpace(value)
		if mapped, ok := shortNames[strings.ToLower(trimmed)]; ok {
			normalized[i] = mapped
		} else {
			normalized[i] = trimmed
		}
	}
	return normalized
}

func resolveAssigneeIDs(ctx core.CommandContext, raw []string) ([]string, error) {
	ids := make([]string, 0, len(raw))

	var members []openapi_client.SuperplaneUsersUser
	membersLoaded := false

	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if _, err := uuid.Parse(trimmed); err == nil {
			ids = append(ids, trimmed)
			continue
		}

		if !membersLoaded {
			var err error
			members, err = listOrganizationMembers(ctx)
			if err != nil {
				return nil, err
			}
			membersLoaded = true
		}

		id, err := findOrganizationMemberIDByEmail(members, trimmed)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func listOrganizationMembers(ctx core.CommandContext) ([]openapi_client.SuperplaneUsersUser, error) {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	response, _, err := ctx.API.UsersAPI.
		UsersListUsers(ctx.Context).
		DomainType(string(core.OrganizationDomainType())).
		DomainId(organizationID).
		Execute()
	if err != nil {
		return nil, err
	}

	return response.GetUsers(), nil
}

func findOrganizationMemberIDByEmail(members []openapi_client.SuperplaneUsersUser, email string) (string, error) {
	var matches []string
	for _, member := range members {
		metadata := member.GetMetadata()
		if strings.EqualFold(metadata.GetEmail(), email) {
			matches = append(matches, metadata.GetId())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("assignee %q not found: not a UUID and no organization member matches that email", email)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("assignee %q is ambiguous: multiple organization members match that email", email)
	}
}
