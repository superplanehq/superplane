package factories

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type orderCreateCommand struct {
	factory     *string
	title       *string
	description *string
	file        *string
	assignees   *[]string
}

func (c *orderCreateCommand) Execute(ctx core.CommandContext) error {
	title := strings.TrimSpace(stringValue(c.title))
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	description, err := c.resolveDescription(ctx)
	if err != nil {
		return err
	}

	factoryID, err := ResolveFactoryID(ctx, stringValue(c.factory))
	if err != nil {
		return err
	}

	assigneeIDs, err := c.resolveAssignees(ctx)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesCreateWorkOrderBody()
	body.SetTitle(title)
	if description != "" {
		body.SetDescription(description)
	}
	if len(assigneeIDs) > 0 {
		body.SetAssigneeIds(assigneeIDs)
	}

	response, _, err := ctx.API.FactoryAPI.
		FactoriesCreateWorkOrder(ctx.Context, factoryID).
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
			"Work order created: %s (state: %s)\nTitle: %s\nAssignees: %s\n",
			order.GetId(),
			formatOrderState(order.GetState()),
			order.GetTitle(),
			formatAssigneeList(order.GetAssignees()),
		)
		return err
	})
}

// resolveDescription returns the work order description, either from
// --description (inline) or from -f/--file (a path, or "-" for stdin).
// The two are mutually exclusive; either may be omitted entirely, leaving
// the work order with no description.
func (c *orderCreateCommand) resolveDescription(ctx core.CommandContext) (string, error) {
	inline := stringValue(c.description)
	filePath := stringValue(c.file)

	if inline != "" && filePath != "" {
		return "", fmt.Errorf("specify only one of --description or --file")
	}
	if inline != "" {
		return inline, nil
	}
	if filePath == "" {
		return "", nil
	}

	var (
		raw []byte
		err error
	)
	if filePath == "-" {
		raw, err = io.ReadAll(ctx.Cmd.InOrStdin())
	} else {
		raw, err = os.ReadFile(filePath) // #nosec G304 -- CLI user-provided path
	}
	if err != nil {
		return "", fmt.Errorf("read description file: %w", err)
	}
	return string(raw), nil
}

// resolveAssignees resolves --assignee values (UUIDs or emails) to user
// UUIDs. When none were given (or all were blank), the order defaults to
// the caller running the command, matching the work order description's
// "if no --assignee is provided, the order will be assigned to the user
// running the command".
func (c *orderCreateCommand) resolveAssignees(ctx core.CommandContext) ([]string, error) {
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
		response, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
		if err != nil {
			return nil, err
		}
		user := response.GetUser()
		id := user.GetId()
		if id == "" {
			return nil, fmt.Errorf("could not determine current user id")
		}
		return []string{id}, nil
	}

	return resolveAssigneeIDs(ctx, trimmed)
}
