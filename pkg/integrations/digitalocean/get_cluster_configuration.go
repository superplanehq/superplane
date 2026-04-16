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

type GetClusterConfiguration struct{}

type GetClusterConfigurationSpec struct {
	DatabaseCluster string `json:"databaseCluster" mapstructure:"databaseCluster"`
}

func (g *GetClusterConfiguration) Name() string {
	return "digitalocean.getClusterConfiguration"
}

func (g *GetClusterConfiguration) Label() string {
	return "Get Cluster Configuration"
}

func (g *GetClusterConfiguration) Description() string {
	return "Retrieve the configuration details for a database cluster"
}

func (g *GetClusterConfiguration) Documentation() string {
	return `The Get Cluster Configuration component retrieves the active configuration for a DigitalOcean Managed Database cluster.

## Use Cases

- **Audit workflows**: Inspect the active cluster configuration for reporting or compliance checks
- **Validation**: Compare the current cluster configuration before updates or maintenance
- **Operational visibility**: Retrieve engine-specific settings that affect behavior and performance

## Configuration

- **Database Cluster**: The managed database cluster to inspect (required)

## Output

Returns the cluster configuration including:
- **databaseClusterId**: The cluster UUID
- **databaseClusterName**: The cluster name
- **config**: The configuration object returned by the DigitalOcean API

## Important Notes

- If you use custom token scopes, this action requires ` + "`database:read`" + `
- The keys inside ` + "`config`" + ` vary by database engine`
}

func (g *GetClusterConfiguration) Icon() string {
	return "info"
}

func (g *GetClusterConfiguration) Color() string {
	return "gray"
}

func (g *GetClusterConfiguration) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetClusterConfiguration) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "databaseCluster",
			Label:       "Database Cluster",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The managed database cluster to inspect",
			Placeholder: "Select a database cluster",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "database_cluster",
				},
			},
		},
	}
}

func (g *GetClusterConfiguration) Setup(ctx core.SetupContext) error {
	spec := GetClusterConfigurationSpec{}
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

func (g *GetClusterConfiguration) Execute(ctx core.ExecutionContext) error {
	spec := GetClusterConfigurationSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	config, err := client.GetDatabaseClusterConfig(spec.DatabaseCluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster configuration: %v", err)
	}

	clusterName := spec.DatabaseCluster
	var metadata DatabaseNodeMetadata
	if ctx.NodeMetadata != nil {
		if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err == nil && metadata.DatabaseClusterName != "" {
			clusterName = metadata.DatabaseClusterName
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.database.cluster.config.fetched",
		[]any{map[string]any{
			"databaseClusterId":   spec.DatabaseCluster,
			"databaseClusterName": clusterName,
			"config":              config,
		}},
	)
}

func (g *GetClusterConfiguration) Cancel(ctx core.ExecutionContext) error { return nil }

func (g *GetClusterConfiguration) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetClusterConfiguration) Actions() []core.Action { return []core.Action{} }

func (g *GetClusterConfiguration) HandleAction(ctx core.ActionContext) error { return nil }

func (g *GetClusterConfiguration) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetClusterConfiguration) Cleanup(ctx core.SetupContext) error { return nil }
