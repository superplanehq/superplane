package cloudsql

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const instancePayloadType = "gcp.cloudsql.instance"

// minDiskSizeGb is the smallest data disk Cloud SQL accepts.
const minDiskSizeGb = 10

func minDiskSizePtr() *int { v := minDiskSizeGb; return &v }

var databaseVersionOptions = []configuration.FieldOption{
	{Label: "PostgreSQL 16", Value: "POSTGRES_16"},
	{Label: "PostgreSQL 15", Value: "POSTGRES_15"},
	{Label: "MySQL 8.0", Value: "MYSQL_8_0"},
	{Label: "SQL Server 2022 Standard", Value: "SQLSERVER_2022_STANDARD"},
}

var editionOptions = []configuration.FieldOption{
	{Label: "Enterprise", Value: "ENTERPRISE"},
	{Label: "Enterprise Plus", Value: "ENTERPRISE_PLUS"},
}

type CreateInstance struct{}

type CreateInstanceSpec struct {
	Name            string `json:"name" mapstructure:"name"`
	DatabaseVersion string `json:"databaseVersion" mapstructure:"databaseVersion"`
	Region          string `json:"region" mapstructure:"region"`
	Tier            string `json:"tier" mapstructure:"tier"`
	DiskSizeGb      int    `json:"diskSizeGb" mapstructure:"diskSizeGb"`
	Edition         string `json:"edition" mapstructure:"edition"`
	RootPassword    string `json:"rootPassword" mapstructure:"rootPassword"`
}

func (c *CreateInstance) Name() string {
	return "gcp.cloudsql.createInstance"
}

func (c *CreateInstance) Label() string {
	return "Cloud SQL • Create Instance"
}

func (c *CreateInstance) Description() string {
	return "Provision a Cloud SQL instance"
}

func (c *CreateInstance) Documentation() string {
	return `The Create Instance component provisions a new Cloud SQL instance.

## Use Cases

- **Environment setup**: Stand up a database server as part of provisioning an environment
- **Ephemeral environments**: Create a dedicated instance for a preview or test run
- **Infrastructure automation**: Provision databases as part of a broader workflow

## Configuration

- **Name**: The instance ID (required)
- **Database Version**: The database engine and version (required)
- **Region**: The region to create the instance in (e.g. ` + "`us-central1`" + `)
- **Tier**: The machine type (e.g. ` + "`db-f1-micro`" + ` for dev/test, ` + "`db-custom-2-7680`" + ` for production)
- **Disk Size (GB)**: The data disk size (minimum 10)
- **Edition**: Enterprise or Enterprise Plus
- **Root Password**: Initial password for the default admin user (optional, stored as a secret)

## Output

Emits a ` + "`gcp.cloudsql.instance`" + ` payload with the instance ` + "`name`" + `, the ` + "`operation`" + ` id, and ` + "`status`" + `.

## Important Notes

- **Instance creation is asynchronous and takes several minutes.** This component returns once the operation is accepted; use **Get Instance** to poll until the instance reaches ` + "`RUNNABLE`" + `.
- Requires the ` + "`roles/cloudsql.admin`" + ` (or ` + "`roles/cloudsql.editor`" + `) IAM role, and the **Cloud SQL Admin API** enabled.`
}

func (c *CreateInstance) Icon() string {
	return "database"
}

func (c *CreateInstance) Color() string {
	return "blue"
}

func (c *CreateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The instance ID for the new Cloud SQL instance",
			Placeholder: "my-instance",
		},
		{
			Name:        "databaseVersion",
			Label:       "Database Version",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "POSTGRES_16",
			Description: "The database engine and version",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: databaseVersionOptions},
			},
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "us-central1",
			Description: "The region to create the instance in",
			Placeholder: "us-central1",
		},
		{
			Name:        "tier",
			Label:       "Tier",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "db-f1-micro",
			Description: "The machine type (e.g. db-f1-micro for dev/test)",
			Placeholder: "db-f1-micro",
		},
		{
			Name:        "diskSizeGb",
			Label:       "Disk Size (GB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     minDiskSizeGb,
			Description: "The data disk size in GB (minimum 10)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{Min: minDiskSizePtr()},
			},
		},
		{
			Name:        "edition",
			Label:       "Edition",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "ENTERPRISE",
			Description: "The Cloud SQL edition",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: editionOptions},
			},
		},
		{
			Name:        "rootPassword",
			Label:       "Root Password",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Initial password for the default admin user (optional)",
		},
	}
}

func (c *CreateInstance) Setup(ctx core.SetupContext) error {
	spec := CreateInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(spec.DatabaseVersion) == "" {
		return fmt.Errorf("databaseVersion is required")
	}
	if strings.TrimSpace(spec.Region) == "" {
		return fmt.Errorf("region is required")
	}
	if strings.TrimSpace(spec.Tier) == "" {
		return fmt.Errorf("tier is required")
	}
	return ctx.Metadata.Set(InstanceNodeMetadata{Instance: strings.TrimSpace(spec.Name)})
}

func (c *CreateInstance) Execute(ctx core.ExecutionContext) error {
	spec := CreateInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ctx.ExecutionState.Fail("error", "name is required")
	}

	// Cloud SQL rejects disks below 10 GB, so clamp anything under the minimum
	// (including the 0 default) up to it rather than forwarding a value the API
	// will reject.
	diskSize := spec.DiskSizeGb
	if diskSize < minDiskSizeGb {
		diskSize = minDiskSizeGb
	}
	settings := map[string]any{
		"tier":           strings.TrimSpace(spec.Tier),
		"dataDiskSizeGb": strconv.Itoa(diskSize),
	}
	if edition := strings.TrimSpace(spec.Edition); edition != "" {
		settings["edition"] = edition
	}
	body := map[string]any{
		"name":            name,
		"region":          strings.TrimSpace(spec.Region),
		"databaseVersion": strings.TrimSpace(spec.DatabaseVersion),
		"settings":        settings,
	}
	if pw := spec.RootPassword; pw != "" {
		body["rootPassword"] = pw
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	op, err := createInstance(context.Background(), client, client.ProjectID(), body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to create instance", err, roleHintAdmin))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, instancePayloadType, []any{
		map[string]any{
			"name":      name,
			"operation": op.Name,
			"status":    op.Status,
			"state":     "PENDING_CREATE",
		},
	})
}

func (c *CreateInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
