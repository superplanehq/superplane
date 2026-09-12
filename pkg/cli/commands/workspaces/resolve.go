package workspaces

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
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
	return trimmed, nil
}

func errWorkspaceRequired() error {
	return fmt.Errorf("workspace is required; pass --workspace or set one with \"superplane workspace active\"")
}

func FindWorkspaceID(ctx core.CommandContext, nameOrID string) (string, error) {
	trimmed := strings.TrimSpace(nameOrID)
	if trimmed == "" {
		return "", fmt.Errorf("workspace key or id is required")
	}
	if _, err := uuid.Parse(trimmed); err == nil {
		return trimmed, nil
	}

	response, _, err := ctx.API.FactoryAPI.FactoriesDescribeFactory(ctx.Context, trimmed).Execute()
	if err != nil {
		return "", err
	}
	if response == nil || !response.HasFactory() {
		return "", fmt.Errorf("workspace %q is missing an id", trimmed)
	}
	factory := response.GetFactory()
	if !factory.HasId() {
		return "", fmt.Errorf("workspace %q is missing an id", trimmed)
	}

	return factory.GetId(), nil
}
