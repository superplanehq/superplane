package tasks

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/workspaces"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
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

func resolveTaskID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("--task is required")
	}
	return trimmed, nil
}

func taskDisplayID(task openapi_client.FactoriesWorkOrder) string {
	if number := strings.TrimSpace(task.GetNumber()); number != "" {
		return number
	}
	if key := strings.TrimSpace(task.GetKey()); key != "" {
		return key
	}
	return task.GetId()
}
