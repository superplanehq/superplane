package common

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// FindAppID returns the automation id for the given name or UUID. If
// nameOrID parses as a UUID, it is returned unchanged. Otherwise the
// CLI looks up automations by name and requires exactly one match.
func FindAppID(ctx core.CommandContext, client *openapi_client.APIClient, nameOrID string) (string, error) {
	if _, err := uuid.Parse(nameOrID); err == nil {
		return nameOrID, nil
	}

	return findAppIDByName(ctx, client, nameOrID)
}

func findAppIDByName(ctx core.CommandContext, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.CanvasesCanvasSummary
	for _, canvas := range response.GetCanvases() {
		if canvas.Name == nil {
			continue
		}
		if *canvas.Name == name {
			matches = append(matches, canvas)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("automation %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple automations named %q found", name)
	}

	if matches[0].Id == nil {
		return "", fmt.Errorf("automation %q is missing an id", name)
	}

	return *matches[0].Id, nil
}
