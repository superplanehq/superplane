package tasks

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/workspaces"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func resolveWorkspace(ctx core.CommandContext, workspaceFlag *string) (string, error) {
	return workspaces.ResolveWorkspaceID(ctx, stringValue(workspaceFlag))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func resolveTaskID(ctx core.CommandContext, workspaceID, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("--task is required")
	}

	if _, err := uuid.Parse(trimmed); err == nil {
		return trimmed, nil
	}

	response, _, err := ctx.API.FactoryAPI.FactoriesListWorkOrders(ctx.Context, workspaceID).Execute()
	if err != nil {
		return "", err
	}

	for _, task := range response.GetOrders() {
		if task.GetKey() == trimmed {
			return task.GetId(), nil
		}
	}

	return "", fmt.Errorf("task %q not found", trimmed)
}
