package mcpserver

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const workspaceArg = "workspace"

// organizationScopedTools act on the organization, not on one workspace, so
// they take no workspace argument.
var organizationScopedTools = []string{"list_workspaces", "list_members"}

func listWorkspacesTool() Tool {
	return Tool{
		Name:        "list_workspaces",
		Description: "List the workspaces this connection can use. Each workspace has an id, a key, and a name. Pass an id or a key as the workspace argument of a task tool.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// withWorkspaceArgument adds the optional workspace argument to every tool that
// acts on one workspace.
func withWorkspaceArgument(tools []Tool) []Tool {
	for _, tool := range tools {
		if slices.Contains(organizationScopedTools, tool.Name) {
			continue
		}
		properties, _ := tool.InputSchema["properties"].(map[string]any)
		if properties == nil {
			continue
		}
		properties[workspaceArg] = map[string]any{
			"type":        "string",
			"description": "Workspace id or key. Required when this connection can use more than one workspace. Call list_workspaces to see the choices.",
		}
	}
	return tools
}

func (rt *Runtime) listWorkspaces(ctx context.Context, claims *AccessClaims, _ map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}

	type workspaceRow struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	db := database.DB(ctx)
	rows := make([]workspaceRow, 0, len(claims.FactoryIDs))
	for _, id := range claims.FactoryIDs {
		factory, err := models.FindFactory(db, claims.OrgID, id)
		if err != nil {
			continue
		}
		rows = append(rows, workspaceRow{ID: factory.ID.String(), Key: factory.Key, Name: factory.Name})
	}
	return TextResult(mustJSON(map[string]any{"workspaces": rows})), nil
}

// resolveWorkspace returns the workspace a tool call acts on. The workspace
// must be one the connection was granted in the connection's organization.
// Without a workspace argument, a connection with one workspace uses it.
func resolveWorkspace(ctx context.Context, claims *AccessClaims, args map[string]any) (uuid.UUID, error) {
	ref, err := optionalStringArg(args, workspaceArg)
	if err != nil {
		return uuid.Nil, err
	}
	if ref == "" {
		if len(claims.FactoryIDs) == 1 {
			return claims.FactoryIDs[0], nil
		}
		return uuid.Nil, ToolError("Set workspace. Call list_workspaces to see the workspaces this connection can use.")
	}

	factory, err := models.FindFactoryByRef(database.DB(ctx), claims.OrgID, ref)
	if err != nil || !slices.Contains(claims.FactoryIDs, factory.ID) {
		return uuid.Nil, ToolError("Workspace not found. Call list_workspaces to see the workspaces this connection can use.")
	}
	return factory.ID, nil
}
