package factories

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// ResolveFactoryID returns the factory id from --factory or the active factory.
// Name values are resolved via ListFactories; UUIDs pass through.
func ResolveFactoryID(ctx core.CommandContext, factoryFlag string) (string, error) {
	trimmed := strings.TrimSpace(factoryFlag)
	if trimmed == "" {
		if ctx.Config == nil {
			return "", errFactoryRequired()
		}
		trimmed = strings.TrimSpace(ctx.Config.GetActiveFactory())
	}
	if trimmed == "" {
		return "", errFactoryRequired()
	}
	return FindFactoryID(ctx, trimmed)
}

func errFactoryRequired() error {
	return fmt.Errorf("factory is required; pass --factory or set one with \"superplane factory active\"")
}

// FindFactoryID returns the factory id for a name or UUID.
// UUID values pass through; names are resolved via ListFactories.
func FindFactoryID(ctx core.CommandContext, nameOrID string) (string, error) {
	trimmed := strings.TrimSpace(nameOrID)
	if trimmed == "" {
		return "", fmt.Errorf("factory name or id is required")
	}
	if _, err := uuid.Parse(trimmed); err == nil {
		return trimmed, nil
	}

	response, _, err := ctx.API.FactoryAPI.FactoriesListFactories(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.FactoriesFactory
	for _, factory := range response.GetFactories() {
		if factory.GetName() == trimmed {
			matches = append(matches, factory)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("factory %q not found", trimmed)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple factories named %q found", trimmed)
	}
	if !matches[0].HasId() {
		return "", fmt.Errorf("factory %q is missing an id", trimmed)
	}

	return matches[0].GetId(), nil
}
