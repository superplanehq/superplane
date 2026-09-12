package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type taskAssignCommand struct {
	workspace *string
	taskID    *string
	assignees *[]string
}

func (c *taskAssignCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := strings.TrimSpace(stringValue(c.taskID))
	if rawTaskID == "" {
		return fmt.Errorf("--task is required")
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

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(rawTaskID)
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
		FactoriesUpdateWorkOrderAssignees(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	task := response.GetOrder()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(task)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(
			stdout,
			"Task assignees updated: %s\nAssignees: %s\n",
			taskDisplayID(task),
			formatAssigneeList(task.GetAssignees()),
		)
		return err
	})
}
