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
	user       *string
	states     *[]string
	results    *[]string
	unassigned *bool
	all        *bool
}

type taskListResult struct {
	tasks     []openapi_client.FactoriesWorkOrderSummary
	truncated bool
}

type workOrderListQuery struct {
	workspaceID string
	userID      string
	states      []string
	results     []string
	unassigned  bool
}

const taskListPageSize int64 = 100

func (c *taskListCommand) Execute(ctx core.CommandContext) error {
	workspaceID, err := workspaces.ResolveWorkspaceID(ctx, stringValue(c.workspace))
	if err != nil {
		return err
	}

	result, err := c.listWorkOrders(ctx, workspaceID)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(result.tasks)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderTaskListText(stdout, result)
	})
}

func renderTaskListText(stdout io.Writer, result taskListResult) error {
	if len(result.tasks) == 0 {
		_, err := fmt.Fprintln(stdout, "No tasks found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "NUMBER\tTITLE\tSTATE\tRESULT\tASSIGNEES\tEXECUTIONS\tCREATED")
	for _, task := range result.tasks {
		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			taskDisplayID(&task),
			task.GetTitle(),
			formatTaskState(task.GetState()),
			formatTaskResult(task.GetResult()),
			formatAssigneeList(task.GetAssignees()),
			countStepExecutions(task.GetLineDispatches()),
			formatRelativeTime(task.GetCreatedAt()),
		)
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if !result.truncated {
		return nil
	}
	_, err := fmt.Fprintf(stdout, "\n%s\n", taskListTruncatedMessage())
	return err
}

func taskListTruncatedMessage() string {
	return fmt.Sprintf(
		"This list shows the first %d matching tasks.\nUse --user, --state, --result, or --unassigned to narrow the list.\nPass --all to show every matching task.",
		taskListPageSize,
	)
}

func (c *taskListCommand) listWorkOrders(
	ctx core.CommandContext,
	workspaceID string,
) (taskListResult, error) {
	query, err := c.newWorkOrderListQuery(ctx, workspaceID)
	if err != nil {
		return taskListResult{}, err
	}
	if c.all != nil && *c.all {
		return query.fetchAllPages(ctx)
	}
	return query.fetchFirstPage(ctx)
}

func (c *taskListCommand) newWorkOrderListQuery(
	ctx core.CommandContext,
	workspaceID string,
) (workOrderListQuery, error) {
	userID, err := c.resolveListUserID(ctx)
	if err != nil {
		return workOrderListQuery{}, err
	}

	var rawStates []string
	if c.states != nil {
		rawStates = *c.states
	}
	states := resolveStateFilter(rawStates)
	if states != nil {
		states = normalizeFilterValues(states, shortTaskStateTokens)
	}

	var results []string
	if c.results != nil && len(*c.results) > 0 {
		results = normalizeFilterValues(*c.results, shortTaskResultTokens)
	}

	return workOrderListQuery{
		workspaceID: workspaceID,
		userID:      userID,
		states:      states,
		results:     results,
		unassigned:  c.unassigned != nil && *c.unassigned,
	}, nil
}

func (q workOrderListQuery) fetchFirstPage(ctx core.CommandContext) (taskListResult, error) {
	page, hasNextPage, err := q.fetchPage(ctx, "")
	if err != nil {
		return taskListResult{}, err
	}
	return taskListResult{tasks: page, truncated: hasNextPage}, nil
}

func (q workOrderListQuery) fetchAllPages(ctx core.CommandContext) (taskListResult, error) {
	var tasks []openapi_client.FactoriesWorkOrderSummary
	var beforeID string
	for {
		page, hasNextPage, err := q.fetchPage(ctx, beforeID)
		if err != nil {
			return taskListResult{}, err
		}
		tasks = append(tasks, page...)
		if !hasNextPage || len(page) == 0 {
			return taskListResult{tasks: tasks}, nil
		}
		nextID := page[len(page)-1].GetId()
		if nextID == "" {
			return taskListResult{tasks: tasks}, nil
		}
		beforeID = nextID
	}
}

func (q workOrderListQuery) fetchPage(
	ctx core.CommandContext,
	beforeID string,
) ([]openapi_client.FactoriesWorkOrderSummary, bool, error) {
	request := ctx.API.FactoryAPI.FactoriesListWorkOrders(ctx.Context, q.workspaceID).Limit(taskListPageSize)
	if q.userID != "" {
		request = request.UserId(q.userID)
	}
	if q.states != nil {
		request = request.States(q.states)
	}
	if len(q.results) > 0 {
		request = request.Results(q.results)
	}
	if q.unassigned {
		request = request.Unassigned(true)
	}
	if beforeID != "" {
		request = request.BeforeId(beforeID)
	}

	response, _, err := request.Execute()
	if err != nil {
		return nil, false, err
	}
	return response.GetOrders(), response.GetHasNextPage(), nil
}

func (c *taskListCommand) resolveListUserID(ctx core.CommandContext) (string, error) {
	if c.user == nil || strings.TrimSpace(*c.user) == "" {
		return "", nil
	}
	userIDs, err := resolveAssigneeIDs(ctx, []string{*c.user})
	if err != nil {
		return "", err
	}
	if len(userIDs) == 0 {
		return "", nil
	}
	return userIDs[0], nil
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
