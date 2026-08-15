package factories

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type orderAssignCommand struct {
	factory   *string
	orderID   *string
	assignees *[]string
}

func (c *orderAssignCommand) Execute(ctx core.CommandContext) error {
	orderID := strings.TrimSpace(stringValue(c.orderID))
	if orderID == "" {
		return fmt.Errorf("--order is required")
	}

	var raw []string
	if c.assignees != nil {
		raw = *c.assignees
	}

	trimmed := make([]string, 0, len(raw))
	for _, value := range raw {
		if v := strings.TrimSpace(value); v != "" {
			trimmed = append(trimmed, v)
		}
	}
	if len(trimmed) == 0 {
		return fmt.Errorf("--assignee is required (at least one); assign replaces the full assignee list")
	}

	factoryID, err := ResolveFactoryID(ctx, stringValue(c.factory))
	if err != nil {
		return err
	}

	assigneeIDs, err := resolveAssigneeIDs(ctx, trimmed)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesUpdateWorkOrderAssigneesBody()
	body.SetAssigneeIds(assigneeIDs)

	response, _, err := ctx.API.FactoryAPI.
		FactoriesUpdateWorkOrderAssignees(ctx.Context, factoryID, orderID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	order := response.GetOrder()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(order)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(
			stdout,
			"Work order assignees updated: %s\nAssignees: %s\n",
			order.GetId(),
			formatAssigneeList(order.GetAssignees()),
		)
		return err
	})
}
