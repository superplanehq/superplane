package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const databaseClusterPollInterval = 30 * time.Second

type CreateDatabaseCluster struct{}

type CreateDatabaseClusterSpec struct {
	Name     string `json:"name" mapstructure:"name"`
	Engine   string `json:"engine" mapstructure:"engine"`
	Version  string `json:"version" mapstructure:"version"`
	Region   string `json:"region" mapstructure:"region"`
	Size     string `json:"size" mapstructure:"size"`
	NumNodes string `json:"numNodes" mapstructure:"numNodes"`
}

func (c *CreateDatabaseCluster) Name() string {
	return "digitalocean.createDatabaseCluster"
}

func (c *CreateDatabaseCluster) Label() string {
	return "Create Database Cluster"
}

func (c *CreateDatabaseCluster) Description() string {
	return "Create a new database cluster"
}

func (c *CreateDatabaseCluster) Documentation() string {
	return `The Create Database Cluster component provisions a new DigitalOcean Managed Database cluster and waits until it is online.

## Use Cases

- **Environment bootstrap**: Provision a managed database cluster before creating apps or databases
- **Platform setup**: Create a dedicated cluster for a service, team, or customer environment
- **Migration workflows**: Stand up a new cluster before importing data or cutover

## Configuration

- **Name**: The database cluster name (required)
- **Engine**: The database engine to provision, such as PostgreSQL or MySQL (required)
- **Version**: The engine version to provision (required)
- **Region**: The DigitalOcean region for the cluster (required)
- **Size**: The node size slug for the cluster, for example ` + "`db-s-1vcpu-1gb`" + ` (required)
- **Node Count**: The number of nodes in the cluster (required)

## Output

Returns the created database cluster including:
- **id**: The cluster UUID
- **name**: The cluster name
- **engine**: The provisioned engine
- **version**: The engine version
- **region**: The cluster region
- **size**: The selected node size slug
- **num_nodes**: The number of nodes
- **status**: The current cluster status
- **connection**: Connection information when available

## Important Notes

- If you use custom token scopes, this action requires ` + "`database:create`" + ` and ` + "`database:read`" + `
- Valid versions, sizes, and node counts depend on the selected engine. Use the DigitalOcean Database Options API or dashboard values when configuring this component
- The component polls until the cluster status becomes ` + "`online`" + ``
}

func (c *CreateDatabaseCluster) Icon() string {
	return "database"
}

func (c *CreateDatabaseCluster) Color() string {
	return "blue"
}

func (c *CreateDatabaseCluster) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDatabaseCluster) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the database cluster",
			Placeholder: "superplane-db",
		},
		{
			Name:        "engine",
			Label:       "Engine",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "pg",
			Description: "The database engine to provision",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "PostgreSQL", Value: "pg"},
						{Label: "MySQL", Value: "mysql"},
						{Label: "MongoDB", Value: "mongodb"},
						{Label: "Kafka", Value: "kafka"},
						{Label: "OpenSearch", Value: "opensearch"},
						{Label: "Redis", Value: "redis"},
						{Label: "Valkey", Value: "valkey"},
					},
				},
			},
		},
		{
			Name:        "version",
			Label:       "Version",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The engine version to provision",
			Placeholder: "Select a version",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "database_cluster_version",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "engine",
							ValueFrom: &configuration.ParameterValueFrom{Field: "engine"},
						},
					},
				},
			},
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a region",
			Description: "The region to deploy the cluster in",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "region",
				},
			},
		},
		{
			Name:        "size",
			Label:       "Size",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The node size slug for the cluster",
			Placeholder: "Select a size",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "database_cluster_size",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "engine",
							ValueFrom: &configuration.ParameterValueFrom{Field: "engine"},
						},
						{
							Name:      "numNodes",
							ValueFrom: &configuration.ParameterValueFrom{Field: "numNodes"},
						},
					},
				},
			},
		},
		{
			Name:        "numNodes",
			Label:       "Node Count",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "1",
			Description: "The number of nodes in the cluster",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "1", Value: "1"},
						{Label: "2", Value: "2"},
						{Label: "3", Value: "3"},
						{Label: "6", Value: "6"},
						{Label: "9", Value: "9"},
						{Label: "15", Value: "15"},
					},
				},
			},
		},
	}
}

func (c *CreateDatabaseCluster) Setup(ctx core.SetupContext) error {
	spec := CreateDatabaseClusterSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}
	if spec.Engine == "" {
		return errors.New("engine is required")
	}
	if spec.Version == "" {
		return errors.New("version is required")
	}
	if spec.Region == "" {
		return errors.New("region is required")
	}
	if spec.Size == "" {
		return errors.New("size is required")
	}
	if spec.NumNodes == "" {
		return errors.New("numNodes is required")
	}
	if _, err := strconv.Atoi(spec.NumNodes); err != nil {
		return fmt.Errorf("invalid numNodes value %q: %v", spec.NumNodes, err)
	}

	return nil
}

func (c *CreateDatabaseCluster) Execute(ctx core.ExecutionContext) error {
	spec := CreateDatabaseClusterSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	numNodes, err := strconv.Atoi(spec.NumNodes)
	if err != nil {
		return fmt.Errorf("invalid numNodes value %q: %v", spec.NumNodes, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	cluster, err := client.CreateDatabaseCluster(CreateDatabaseClusterRequest{
		Name:     spec.Name,
		Engine:   spec.Engine,
		Version:  spec.Version,
		Region:   spec.Region,
		Size:     spec.Size,
		NumNodes: numNodes,
	})
	if err != nil {
		return fmt.Errorf("failed to create database cluster: %v", err)
	}

	if cluster.Status == "online" {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.database.cluster.created",
			[]any{cluster},
		)
	}

	if err := ctx.Metadata.Set(map[string]any{"databaseClusterID": cluster.ID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, databaseClusterPollInterval)
}

func (c *CreateDatabaseCluster) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *CreateDatabaseCluster) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *CreateDatabaseCluster) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateDatabaseCluster) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		DatabaseClusterID string `mapstructure:"databaseClusterID"`
	}

	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if metadata.DatabaseClusterID == "" {
		return errors.New("database cluster ID is missing from metadata")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	cluster, err := client.GetDatabaseCluster(metadata.DatabaseClusterID)
	if err != nil {
		return fmt.Errorf("failed to get database cluster: %v", err)
	}

	switch cluster.Status {
	case "online":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.database.cluster.created",
			[]any{cluster},
		)
	case "failed":
		return errors.New("database cluster reached failed status")
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, databaseClusterPollInterval)
	}
}

func (c *CreateDatabaseCluster) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *CreateDatabaseCluster) Cleanup(ctx core.SetupContext) error { return nil }
