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

type DeleteDatabase struct{}

type DeleteDatabaseSpec struct {
	Instance string `json:"instance" mapstructure:"instance"`
	Database string `json:"database" mapstructure:"database"`
}

func (d *DeleteDatabase) Name() string {
	return "gcp.cloudsql.deleteDatabase"
}

func (d *DeleteDatabase) Label() string {
	return "Cloud SQL • Delete Database"
}

func (d *DeleteDatabase) Description() string {
	return "Delete a logical database from a Cloud SQL instance"
}

func (d *DeleteDatabase) Documentation() string {
	return `The Delete Database component permanently deletes a logical database from a Cloud SQL instance.

## Use Cases

- **Teardown**: Remove a database as part of decommissioning an environment
- **Tenant offboarding**: Delete a customer's dedicated database
- **Cleanup**: Drop temporary databases created during a workflow

## Configuration

- **Instance**: The Cloud SQL instance that contains the database (required)
- **Database**: The database to delete (required)

## Output

Emits a ` + "`gcp.cloudsql.database`" + ` payload with the deleted database's ` + "`name`" + ` and ` + "`instance`" + `, and ` + "`deleted: true`" + `.

## Important Notes

- **This permanently deletes the database and all its data — it is irreversible.**
- Requires the ` + "`roles/cloudsql.admin`" + ` (or ` + "`roles/cloudsql.editor`" + `) IAM role on the integration's service account, and the **Cloud SQL Admin API** enabled
- Cloud SQL database deletion is asynchronous; this component waits for the operation to finish before emitting`
}

func (d *DeleteDatabase) Icon() string {
	return "database"
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
			Description: "The database to delete",
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

func (d *DeleteDatabase) Setup(ctx core.SetupContext) error {
	spec := DeleteDatabaseSpec{}
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

func (d *DeleteDatabase) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDatabaseSpec{}
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

	if err := deleteDatabase(context.Background(), client, client.ProjectID(), instance, database); err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to delete database", err, roleHintAdmin))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.cloudsql.database", []any{
		map[string]any{
			"name":     database,
			"instance": instance,
			"deleted":  true,
		},
	})
}

func (d *DeleteDatabase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteDatabase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteDatabase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteDatabase) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteDatabase) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteDatabase) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
