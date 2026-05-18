package apps

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

// findAppID resolves an app name-or-ID to an ID.
// If nameOrID is already a valid UUID, it is returned as-is.
// Otherwise the app list is searched by display name or slug.
func findAppID(ctx core.CommandContext, nameOrID string) (string, error) {
	if _, err := uuid.Parse(nameOrID); err == nil {
		return nameOrID, nil
	}

	return findAppIDByNameOrSlug(ctx, nameOrID)
}

func findAppIDByNameOrSlug(ctx core.CommandContext, nameOrSlug string) (string, error) {
	response, _, err := ctx.API.AppAPI.AppsListApps(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matchID string
	var matchCount int

	for _, app := range response.GetApps() {
		metadata := app.GetMetadata()

		if metadata.GetDisplayName() == nameOrSlug || metadata.GetSlug() == nameOrSlug {
			matchID = metadata.GetId()
			matchCount++
		}
	}

	if matchCount == 0 {
		return "", fmt.Errorf("app %q not found", nameOrSlug)
	}

	if matchCount > 1 {
		return "", fmt.Errorf("multiple apps matching %q found; use the app ID instead", nameOrSlug)
	}

	return matchID, nil
}
