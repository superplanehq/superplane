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

type CreateDatabase struct{}

type CreateDatabaseSpec struct {
	Instance string `json:"instance" mapstructure:"instance"`
	Name     string `json:"name" mapstructure:"name"`
}

func (c *CreateDatabase) Name() string {
	return "gcp.cloudsql.createDatabase"
}

func (c *CreateDatabase) Label() string {
	return "Cloud SQL • Create Database"
}

func (c *CreateDatabase) Description() string {
	return "Create a logical database inside a Cloud SQL instance"
}

func (c *CreateDatabase) Documentation() string {
	return `The Create Database component adds a new logical database to an existing Cloud SQL instance.

## Use Cases

- **Application bootstrap**: Create an application-specific database as part of environment setup
- **Tenant provisioning**: Add a dedicated database for a new customer or workspace
- **Migration workflows**: Prepare a destination database before importing data

## Configuration

- **Instance**: The Cloud SQL instance that will contain the new database (required)
- **Database Name**: The name of the database to create (required, supports expressions)

## Output

Emits a ` + "`gcp.cloudsql.database`" + ` payload with the created database's ` + "`name`" + `, ` + "`instance`" + `, ` + "`project`" + `, ` + "`charset`" + `, ` + "`collation`" + `, and ` + "`selfLink`" + `.

## Important Notes

- Requires the ` + "`roles/cloudsql.admin`" + ` (or ` + "`roles/cloudsql.editor`" + `) IAM role on the integration's service account, and the **Cloud SQL Admin API** enabled
- Cloud SQL database creation is asynchronous; this component waits for the operation to finish before emitting`
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
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud SQL instance to create the database in",
			Placeholder: "Select an instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
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
	if strings.TrimSpace(spec.Instance) == "" {
		return fmt.Errorf("instance is required")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}
	return ctx.Metadata.Set(DatabaseNodeMetadata{
		Instance: strings.TrimSpace(spec.Instance),
		Database: strings.TrimSpace(spec.Name),
	})
}

func (c *CreateDatabase) Execute(ctx core.ExecutionContext) error {
	spec := CreateDatabaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	instance := strings.TrimSpace(spec.Instance)
	name := strings.TrimSpace(spec.Name)
	if instance == "" {
		return ctx.ExecutionState.Fail("error", "instance is required")
	}
	if name == "" {
		return ctx.ExecutionState.Fail("error", "name is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	db, err := createDatabase(context.Background(), client, client.ProjectID(), instance, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to create database", err, roleHintAdmin))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.cloudsql.database", []any{databasePayload(db)})
}

func (c *CreateDatabase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDatabase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDatabase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateDatabase) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateDatabase) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateDatabase) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
