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

type CreateDatabase struct{}

type CreateDatabaseSpec struct {
	DatabaseCluster string `json:"databaseCluster" mapstructure:"databaseCluster"`
	Name            string `json:"name" mapstructure:"name"`
}

func (c *CreateDatabase) Name() string {
	return "digitalocean.createDatabase"
}

func (c *CreateDatabase) Label() string {
	return "Create Database"
}

func (c *CreateDatabase) Description() string {
	return "Create a new managed database instance in DigitalOcean"
}

func (c *CreateDatabase) Documentation() string {
	return `The Create Database component adds a new database to an existing DigitalOcean Managed Database cluster.

## Use Cases

- **Application bootstrap**: Create an application-specific database as part of environment setup
- **Tenant provisioning**: Add a dedicated database for a new customer or workspace
- **Migration workflows**: Prepare a destination database before importing data

## Configuration

- **Database Cluster**: The managed database cluster that will contain the new database (required)
- **Database Name**: The name of the database to create (required, supports expressions)

## Output

Returns the created database including:
- **name**: The created database name
- **databaseClusterId**: The cluster UUID
- **databaseClusterName**: The cluster name

## Important Notes

- If you use custom token scopes, this action requires ` + "`database:create`" + ` and ` + "`database:read`" + `
- Database management is not supported for Caching or Valkey clusters`
}

func (c *CreateDatabase) Icon() string {
	return "database"
}

func (c *CreateDatabase) Color() string {
	return "blue"
}

func (c *CreateDatabase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDatabase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "databaseCluster",
			Label:       "Database Cluster",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The managed database cluster to create the database in",
			Placeholder: "Select a database cluster",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "database_cluster",
				},
			},
		},
		{
			Name:        "name",
			Label:       "Database Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the database to create",
			Placeholder: "app_db",
		},
	}
}

func (c *CreateDatabase) Setup(ctx core.SetupContext) error {
	spec := CreateDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DatabaseCluster == "" {
		return errors.New("databaseCluster is required")
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if err := resolveDatabaseClusterMetadata(ctx, spec.DatabaseCluster); err != nil {
		return fmt.Errorf("error resolving database cluster metadata: %v", err)
	}

	var metadata DatabaseNodeMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	metadata.DatabaseName = spec.Name

	return ctx.Metadata.Set(metadata)
}

func (c *CreateDatabase) Execute(ctx core.ExecutionContext) error {
	spec := CreateDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	database, err := client.CreateDatabase(spec.DatabaseCluster, CreateDatabaseRequest{Name: spec.Name})
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}

	clusterName := spec.DatabaseCluster
	var metadata DatabaseNodeMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err == nil && metadata.DatabaseClusterName != "" {
		clusterName = metadata.DatabaseClusterName
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.database.created",
		[]any{map[string]any{
			"name":                database.Name,
			"databaseClusterId":   spec.DatabaseCluster,
			"databaseClusterName": clusterName,
		}},
	)
}

func (c *CreateDatabase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDatabase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDatabase) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateDatabase) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateDatabase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateDatabase) Cleanup(ctx core.SetupContext) error {
	return nil
}
