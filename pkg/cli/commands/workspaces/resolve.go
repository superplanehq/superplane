package workspaces

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func ResolveWorkspaceID(ctx core.CommandContext, workspaceFlag string) (string, error) {
	trimmed := strings.TrimSpace(workspaceFlag)
	if trimmed == "" {
		if ctx.Config == nil {
			return "", errWorkspaceRequired()
		}
		trimmed = strings.TrimSpace(ctx.Config.GetActiveWorkspace())
	}
	if trimmed == "" {
		return "", errWorkspaceRequired()
	}
	return FindWorkspaceID(ctx, trimmed)
}

func errWorkspaceRequired() error {
	return fmt.Errorf("workspace is required; pass --workspace or set one with \"superplane workspace active\"")
}

func FindWorkspaceID(ctx core.CommandContext, nameOrID string) (string, error) {
	trimmed := strings.TrimSpace(nameOrID)
	if trimmed == "" {
		return "", fmt.Errorf("workspace name or id is required")
	}
	if _, err := uuid.Parse(trimmed); err == nil {
		return trimmed, nil
	}

	response, _, err := ctx.API.FactoryAPI.FactoriesListFactories(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.FactoriesFactory
	for _, ws := range response.GetFactories() {
		if ws.GetName() == trimmed {
			matches = append(matches, ws)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("workspace %q not found", trimmed)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple workspaces named %q found", trimmed)
	}
	if !matches[0].HasId() {
		return "", fmt.Errorf("workspace %q is missing an id", trimmed)
	}

	return matches[0].GetId(), nil
}
