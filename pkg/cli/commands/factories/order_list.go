package factories

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type orderListCommand struct {
	factory    *string
	assignees  *[]string
	states     *[]string
	results    *[]string
	unassigned *bool
}

func (c *orderListCommand) Execute(ctx core.CommandContext) error {
	factoryID, err := ResolveFactoryID(ctx, stringValue(c.factory))
	if err != nil {
		return err
	}

	request := ctx.API.FactoryAPI.FactoriesListWorkOrders(ctx.Context, factoryID)

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
		request = request.States(normalizeFilterValues(resolved, shortOrderStateTokens))
	}

	if c.results != nil && len(*c.results) > 0 {
		request = request.Results(normalizeFilterValues(*c.results, shortOrderResultTokens))
	}

	if c.unassigned != nil && *c.unassigned {
		request = request.Unassigned(true)
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	orders := response.GetOrders()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(orders)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(orders) == 0 {
			_, err := fmt.Fprintln(stdout, "No work orders found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tTITLE\tSTATE\tRESULT\tASSIGNEES\tEXECUTIONS\tCREATED")
		for _, order := range orders {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
				order.GetId(),
				order.GetTitle(),
				formatOrderState(order.GetState()),
				formatOrderResult(order.GetResult()),
				formatAssigneeList(order.GetAssignees()),
				countStepExecutions(order.GetLineDispatches()),
				formatRelativeTime(order.GetCreatedAt()),
			)
		}
		return writer.Flush()
	})
}

// countStepExecutions sums the step executions across every line dispatch,
// matching the flat "executions" count the EXECUTIONS column showed before
// line dispatches became first-class aggregates.
func countStepExecutions(dispatches []openapi_client.FactoriesWorkOrderLineDispatch) int {
	total := 0
	for _, dispatch := range dispatches {
		total += len(dispatch.GetStepExecutions())
	}
	return total
}

// shortOrderStateTokens maps short, case-insensitive state names to the
// full proto enum token expected by the API, so --state is pleasant to
// type interactively (e.g. "open" instead of "STATE_OPEN").
var shortOrderStateTokens = map[string]string{
	"intake": "STATE_INTAKE",
	"draft":  "STATE_DRAFT",
	"open":   "STATE_OPEN",
	"closed": "STATE_CLOSED",
}

// orderStateAllToken is the special --state value that opts out of the
// default "open only" filter and restores the "no filter" behavior,
// returning work orders in every state.
const orderStateAllToken = "all"

// defaultOrderStates is applied when --state is omitted entirely, so
// "superplane factory orders list" only shows open work orders by default.
var defaultOrderStates = []string{"open"}

// resolveStateFilter decides which state values (if any) should be sent to
// the API as the "states" filter:
//   - if the caller explicitly passed "all" (case-insensitively, anywhere in
//     the list), no filter is applied at all (nil is returned), restoring the
//     legacy "every state" behavior.
//   - if no states were provided, the default of "open only" is used.
//   - otherwise, the caller's explicit values are returned unchanged (to be
//     normalized by normalizeFilterValues as usual).
func resolveStateFilter(states []string) []string {
	for _, state := range states {
		if strings.EqualFold(strings.TrimSpace(state), orderStateAllToken) {
			return nil
		}
	}

	if len(states) == 0 {
		return defaultOrderStates
	}

	return states
}

// shortOrderResultTokens maps short, case-insensitive result names to the
// full proto enum token expected by the API (e.g. "completed" instead of
// "RESULT_COMPLETED").
var shortOrderResultTokens = map[string]string{
	"completed": "RESULT_COMPLETED",
	"rejected":  "RESULT_REJECTED",
	"failed":    "RESULT_FAILED",
}

// normalizeFilterValues maps each value through the short-name lookup table,
// passing unknown values through unchanged so the server can reject them
// with a clear error rather than the CLI silently dropping them.
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

// resolveAssigneeIDs converts CLI-supplied assignee identifiers (UUIDs or
// emails) into the user UUIDs the API expects. Values that already parse as
// a UUID pass straight through; everything else is resolved against the
// organization's members by email.
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
