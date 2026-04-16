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

type GetDatabaseCluster struct{}

type GetDatabaseClusterSpec struct {
	DatabaseCluster string `json:"databaseCluster" mapstructure:"databaseCluster"`
}

func (g *GetDatabaseCluster) Name() string {
	return "digitalocean.getDatabaseCluster"
}

func (g *GetDatabaseCluster) Label() string {
	return "Get Database Cluster"
}

func (g *GetDatabaseCluster) Description() string {
	return "Retrieve details of a specific database cluster"
}

func (g *GetDatabaseCluster) Documentation() string {
	return `The Get Database Cluster component retrieves the details of an existing DigitalOcean Managed Database cluster.

## Use Cases

- **Status checks**: Verify a cluster is online before creating databases or users
- **Information retrieval**: Fetch connection details, sizing, engine, and region information
- **Pre-flight validation**: Confirm a cluster exists before downstream operations

## Configuration

- **Database Cluster**: The managed database cluster to retrieve (required)

## Output

Returns the database cluster including:
- **id**: The cluster UUID
- **name**: The cluster name
- **engine**: The configured engine
- **version**: The engine version
- **region**: The cluster region
- **size**: The node size slug
- **num_nodes**: The number of nodes
- **status**: The cluster status
- **connection**: Connection information when available

## Important Notes

- If you use custom token scopes, this action requires ` + "`database:read`" + `
- The returned connection information depends on the cluster type and provisioning state.`
}

func (g *GetDatabaseCluster) Icon() string {
	return "info"
}

func (g *GetDatabaseCluster) Color() string {
	return "gray"
}

func (g *GetDatabaseCluster) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDatabaseCluster) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "databaseCluster",
			Label:       "Database Cluster",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The database cluster to retrieve",
			Placeholder: "Select a database cluster",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "database_cluster",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (g *GetDatabaseCluster) Setup(ctx core.SetupContext) error {
	spec := GetDatabaseClusterSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.DatabaseCluster == "" {
		return errors.New("databaseCluster is required")
	}

	if err := resolveDatabaseClusterMetadata(ctx, spec.DatabaseCluster); err != nil {
		return fmt.Errorf("error resolving database cluster metadata: %v", err)
	}

	return nil
}

func (g *GetDatabaseCluster) Execute(ctx core.ExecutionContext) error {
	spec := GetDatabaseClusterSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	cluster, err := client.GetDatabaseCluster(spec.DatabaseCluster)
	if err != nil {
		return fmt.Errorf("failed to get database cluster: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.database.cluster.fetched",
		[]any{cluster},
	)
}

func (g *GetDatabaseCluster) Cancel(ctx core.ExecutionContext) error { return nil }
func (g *GetDatabaseCluster) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (g *GetDatabaseCluster) Actions() []core.Action                    { return []core.Action{} }
func (g *GetDatabaseCluster) HandleAction(ctx core.ActionContext) error { return nil }
func (g *GetDatabaseCluster) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (g *GetDatabaseCluster) Cleanup(ctx core.SetupContext) error { return nil }
