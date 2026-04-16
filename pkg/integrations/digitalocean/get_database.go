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

type GetDatabase struct{}

type GetDatabaseSpec struct {
	DatabaseCluster string `json:"databaseCluster" mapstructure:"databaseCluster"`
	Database        string `json:"database" mapstructure:"database"`
}

func (g *GetDatabase) Name() string {
	return "digitalocean.getDatabase"
}

func (g *GetDatabase) Label() string {
	return "Get Database"
}

func (g *GetDatabase) Description() string {
	return "Retrieve details of a specific managed database instance"
}

func (g *GetDatabase) Documentation() string {
	return `The Get Database component retrieves a managed database from a DigitalOcean cluster and enriches it with cluster context.

## Use Cases

- **Routing decisions**: Inspect the database and cluster state before directing traffic or jobs
- **Operational checks**: Review engine, region, and connection details before maintenance steps
- **Audit workflows**: Retrieve the current database and cluster context for reporting or validation

## Configuration

- **Database Cluster**: The managed database cluster containing the database (required)
- **Database**: The database to retrieve (required)

## Output

Returns the requested database enriched with cluster details, including:
- **name**: The database name
- **databaseClusterId**: The cluster UUID
- **databaseClusterName**: The cluster name
- **engine**: The cluster engine
- **version**: The cluster engine version
- **region**: The cluster region
- **status**: The cluster status
- **connection**: Connection information when available
- **database**: The raw database object returned by the API

## Important Notes

- If you use custom token scopes, this action requires ` + "`database:read`" + `
- Database management is not supported for Caching or Valkey clusters`
}

func (g *GetDatabase) Icon() string {
	return "info"
}

func (g *GetDatabase) Color() string {
	return "gray"
}

func (g *GetDatabase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDatabase) Configuration() []configuration.Field {
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
			Description: "The database to retrieve",
			Placeholder: "Select a database",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "database",
					UseNameAsValue: false,
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

func (g *GetDatabase) Setup(ctx core.SetupContext) error {
	spec := GetDatabaseSpec{}
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

func (g *GetDatabase) Execute(ctx core.ExecutionContext) error {
	spec := GetDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	database, err := client.GetDatabase(spec.DatabaseCluster, spec.Database)
	if err != nil {
		return fmt.Errorf("failed to get database: %v", err)
	}

	cluster, err := client.GetDatabaseCluster(spec.DatabaseCluster)
	if err != nil {
		return fmt.Errorf("failed to get database cluster: %v", err)
	}

	clusterName := cluster.Name
	if clusterName == "" {
		clusterName = spec.DatabaseCluster
	}

	name := spec.Database
	if value, ok := database["name"].(string); ok && value != "" {
		name = value
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.database.fetched",
		[]any{map[string]any{
			"name":                name,
			"databaseClusterId":   spec.DatabaseCluster,
			"databaseClusterName": clusterName,
			"engine":              cluster.Engine,
			"version":             cluster.Version,
			"region":              cluster.Region,
			"status":              cluster.Status,
			"connection":          cluster.Connection,
			"database":            database,
		}},
	)
}

func (g *GetDatabase) Cancel(ctx core.ExecutionContext) error { return nil }

func (g *GetDatabase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetDatabase) Actions() []core.Action { return []core.Action{} }

func (g *GetDatabase) HandleAction(ctx core.ActionContext) error { return nil }

func (g *GetDatabase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetDatabase) Cleanup(ctx core.SetupContext) error { return nil }
