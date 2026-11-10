package tasks

import (
	"fmt"
	"os"
	"strings"

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

// TaskIDEnv names the task that a command uses when --task is empty.
// SuperPlane sets it in the environment of each task run.
const TaskIDEnv = "SUPERPLANE_TASK_ID"

var errTaskRequired = fmt.Errorf("--task is required (or set %s)", TaskIDEnv)

// taskRef returns the --task value, or SUPERPLANE_TASK_ID when --task is empty.
func taskRef(flag *string) string {
	if ref := strings.TrimSpace(stringValue(flag)); ref != "" {
		return ref
	}
	return strings.TrimSpace(os.Getenv(TaskIDEnv))
}

func resolveTaskID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errTaskRequired
	}
	return trimmed, nil
}

type taskIdentity interface {
	GetId() string
	GetNumber() string
	GetKey() string
}

func taskDisplayID(task taskIdentity) string {
	if number := strings.TrimSpace(task.GetNumber()); number != "" {
		return number
	}
	if key := strings.TrimSpace(task.GetKey()); key != "" {
		return key
	}
	return task.GetId()
}
