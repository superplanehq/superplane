package tasks

import "github.com/superplanehq/superplane/pkg/cli/commands/workspaces"
import "github.com/superplanehq/superplane/pkg/cli/core"

func resolveWorkspace(ctx core.CommandContext, workspaceFlag *string) (string, error) {
	return workspaces.ResolveWorkspaceID(ctx, stringValue(workspaceFlag))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
