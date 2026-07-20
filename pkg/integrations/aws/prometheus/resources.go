package prometheus

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListWorkspaces(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	alias := strings.TrimSpace(ctx.Parameters["alias"])
	client := NewClient(ctx.HTTP, creds, region)
	workspaces, err := client.ListWorkspaces(alias)
	if err != nil {
		return nil, fmt.Errorf("failed to list Prometheus workspaces: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(workspaces))
	for _, workspace := range workspaces {
		name := strings.TrimSpace(workspace.Alias)
		if name == "" {
			name = workspace.WorkspaceID
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   workspace.WorkspaceID,
		})
	}

	return resources, nil
}
