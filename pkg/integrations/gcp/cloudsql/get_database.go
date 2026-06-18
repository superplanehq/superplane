package cloudsql

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetDatabase struct{}

type GetDatabaseSpec struct {
	Instance string `json:"instance" mapstructure:"instance"`
	Database string `json:"database" mapstructure:"database"`
}

func (g *GetDatabase) Name() string {
	return "gcp.cloudsql.getDatabase"
}

func (g *GetDatabase) Label() string {
	return "Cloud SQL • Get Database"
}

func (g *GetDatabase) Description() string {
	return "Fetch a logical database from a Cloud SQL instance"
}

func (g *GetDatabase) Documentation() string {
	return `The Get Database component retrieves a logical database from a Cloud SQL instance.

## Use Cases

- **Existence checks**: Confirm a database is present before acting on it
- **Enrichment**: Read a database's charset/collation to feed a downstream step
- **Auditing**: Capture database details as part of a workflow

## Configuration

- **Instance**: The Cloud SQL instance that contains the database (required)
- **Database**: The database to fetch (required)

## Output

Emits a ` + "`gcp.cloudsql.database`" + ` payload with the database's ` + "`name`" + `, ` + "`instance`" + `, ` + "`project`" + `, ` + "`charset`" + `, ` + "`collation`" + `, and ` + "`selfLink`" + `.

## Important Notes

- Requires the ` + "`roles/cloudsql.viewer`" + ` (or ` + "`roles/cloudsql.admin`" + `) IAM role on the integration's service account, and the **Cloud SQL Admin API** enabled`
}

func (g *GetDatabase) Icon() string {
	return "database"
}

func (g *GetDatabase) Color() string {
	return "blue"
}

func (g *GetDatabase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDatabase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud SQL instance that contains the database",
			Placeholder: "Select an instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "database",
			Label:       "Database",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The database to fetch",
			Placeholder: "Select a database",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDatabase,
					Parameters: []configuration.ParameterRef{
						{Name: "instance", ValueFrom: &configuration.ParameterValueFrom{Field: "instance"}},
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
	if strings.TrimSpace(spec.Instance) == "" {
		return fmt.Errorf("instance is required")
	}
	if strings.TrimSpace(spec.Database) == "" {
		return fmt.Errorf("database is required")
	}
	return ctx.Metadata.Set(DatabaseNodeMetadata{
		Instance: strings.TrimSpace(spec.Instance),
		Database: strings.TrimSpace(spec.Database),
	})
}

func (g *GetDatabase) Execute(ctx core.ExecutionContext) error {
	spec := GetDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	instance := strings.TrimSpace(spec.Instance)
	database := strings.TrimSpace(spec.Database)
	if instance == "" {
		return ctx.ExecutionState.Fail("error", "instance is required")
	}
	if database == "" {
		return ctx.ExecutionState.Fail("error", "database is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	db, err := getDatabase(context.Background(), client, client.ProjectID(), instance, database)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to get database", err, roleHintViewer))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.cloudsql.database", []any{databasePayload(db)})
}

func (g *GetDatabase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetDatabase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetDatabase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetDatabase) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetDatabase) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetDatabase) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
