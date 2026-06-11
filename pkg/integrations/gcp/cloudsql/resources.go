package cloudsql

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

// ResourceTypeInstance lists the Cloud SQL instances in the project.
const ResourceTypeInstance = "cloudsqlInstance"

// ListInstanceResources lists Cloud SQL instances for the instance dropdown.
func ListInstanceResources(ctx context.Context, client Client) ([]core.IntegrationResource, error) {
	instances, err := ListInstances(ctx, client, client.ProjectID())
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(instances))
	for _, inst := range instances {
		label := inst.Name
		if inst.DatabaseVersion != "" {
			label = fmt.Sprintf("%s (%s)", inst.Name, inst.DatabaseVersion)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeInstance, Name: label, ID: inst.Name})
	}
	return out, nil
}
