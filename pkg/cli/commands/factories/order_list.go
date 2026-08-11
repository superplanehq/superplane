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

	if c.states != nil && len(*c.states) > 0 {
		request = request.States(normalizeFilterValues(*c.states, shortOrderStateTokens))
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
				len(order.GetExecutions()),
				formatRelativeTime(order.GetCreatedAt()),
			)
		}
		return writer.Flush()
	})
}

// shortOrderStateTokens maps short, case-insensitive state names to the
// full proto enum token expected by the API, so --state is pleasant to
// type interactively (e.g. "open" instead of "STATE_OPEN").
var shortOrderStateTokens = map[string]string{
	"draft":  "STATE_DRAFT",
	"open":   "STATE_OPEN",
	"closed": "STATE_CLOSED",
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
