package cloudsql

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// ResourceTypeInstance lists the Cloud SQL instances in the project.
	ResourceTypeInstance = "cloudsqlInstance"
	// ResourceTypeDatabase lists the databases in a selected instance.
	ResourceTypeDatabase = "cloudsqlDatabase"
)

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

// ListDatabaseResources lists the databases in the selected instance for the
// database dropdown.
func ListDatabaseResources(ctx context.Context, client Client, instance string) ([]core.IntegrationResource, error) {
	if instance == "" {
		return nil, nil
	}
	databases, err := ListDatabases(ctx, client, client.ProjectID(), instance)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(databases))
	for _, db := range databases {
		out = append(out, core.IntegrationResource{Type: ResourceTypeDatabase, Name: db.Name, ID: db.Name})
	}
	return out, nil
}
