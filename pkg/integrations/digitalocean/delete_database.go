package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteDatabase struct{}

type DeleteDatabaseSpec struct {
	DatabaseCluster string `json:"databaseCluster" mapstructure:"databaseCluster"`
	Database        string `json:"database" mapstructure:"database"`
}

func (d *DeleteDatabase) Name() string {
	return "digitalocean.deleteDatabase"
}

func (d *DeleteDatabase) Label() string {
	return "Delete Database"
}

func (d *DeleteDatabase) Description() string {
	return "Delete an existing database instance from DigitalOcean"
}

func (d *DeleteDatabase) Documentation() string {
	return `The Delete Database component permanently removes a database from a DigitalOcean Managed Database cluster.

## Use Cases

- **Cleanup**: Remove databases that are no longer needed after a workflow completes
- **Environment teardown**: Delete temporary or preview-environment databases
- **Tenant offboarding**: Remove customer-specific databases during deprovisioning

## Configuration

- **Database Cluster**: The managed database cluster containing the database (required)
- **Database**: The database to delete (required)

## Output

Returns information about the deleted database:
- **name**: The deleted database name
- **databaseClusterId**: The cluster UUID
- **databaseClusterName**: The cluster name
- **deleted**: Whether the delete request succeeded

## Important Notes

- If you use custom token scopes, this action requires ` + "`database:delete`" + ` and ` + "`database:read`" + `
- Database management is not supported for Caching or Valkey clusters
- Deleting a database that no longer exists is treated as a success`
}

func (d *DeleteDatabase) Icon() string {
	return "trash-2"
}

func (d *DeleteDatabase) Color() string {
	return "red"
}

func (d *DeleteDatabase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteDatabase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "databaseCluster",
			Label:       "Database Cluster",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The managed database cluster containing the database",
			Placeholder: "Select a database cluster",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "database_cluster",
				},
			},
		},
		{
			Name:        "database",
			Label:       "Database",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The database to delete",
			Placeholder: "Select a database",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "database",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "databaseCluster",
							ValueFrom: &configuration.ParameterValueFrom{Field: "databaseCluster"},
						},
					},
				},
			},
		},
	}
}

func (d *DeleteDatabase) Setup(ctx core.SetupContext) error {
	spec := DeleteDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DatabaseCluster == "" {
		return errors.New("databaseCluster is required")
	}

	if spec.Database == "" {
		return errors.New("database is required")
	}

	if err := resolveDatabaseMetadata(ctx, spec.DatabaseCluster, spec.Database); err != nil {
		return fmt.Errorf("error resolving database metadata: %v", err)
	}

	return nil
}

func (d *DeleteDatabase) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteDatabase(spec.DatabaseCluster, spec.Database)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			return emitDeletedDatabase(ctx, spec.DatabaseCluster, spec.Database)
		}
		return fmt.Errorf("failed to delete database: %v", err)
	}

	return emitDeletedDatabase(ctx, spec.DatabaseCluster, spec.Database)
}

func emitDeletedDatabase(ctx core.ExecutionContext, clusterID, databaseName string) error {
	clusterName := clusterID
	var metadata DatabaseNodeMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err == nil && metadata.DatabaseClusterName != "" {
		clusterName = metadata.DatabaseClusterName
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.database.deleted",
		[]any{map[string]any{
			"name":                databaseName,
			"databaseClusterId":   clusterID,
			"databaseClusterName": clusterName,
			"deleted":             true,
		}},
	)
}

func (d *DeleteDatabase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteDatabase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteDatabase) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteDatabase) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteDatabase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteDatabase) Cleanup(ctx core.SetupContext) error {
	return nil
}
