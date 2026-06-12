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
	Name               string       `json:"name" mapstructure:"name"`
	DatabaseVersion    string       `json:"databaseVersion" mapstructure:"databaseVersion"`
	Region             string       `json:"region" mapstructure:"region"`
	Tier               string       `json:"tier" mapstructure:"tier"`
	DiskSizeGb         int          `json:"diskSizeGb" mapstructure:"diskSizeGb"`
	DataDiskType       string       `json:"dataDiskType" mapstructure:"dataDiskType"`
	Edition            string       `json:"edition" mapstructure:"edition"`
	RootPassword       string       `json:"rootPassword" mapstructure:"rootPassword"`
	AvailabilityType   string       `json:"availabilityType" mapstructure:"availabilityType"`
	AutomatedBackups   *bool        `json:"automatedBackups" mapstructure:"automatedBackups"`
	PublicIP           *bool        `json:"publicIp" mapstructure:"publicIp"`
	SSLMode            string       `json:"sslMode" mapstructure:"sslMode"`
	AuthorizedNetworks []string     `json:"authorizedNetworks" mapstructure:"authorizedNetworks"`
	DeletionProtection *bool        `json:"deletionProtection" mapstructure:"deletionProtection"`
	Labels             []labelEntry `json:"labels" mapstructure:"labels"`
}

type labelEntry struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

var sslModeOptions = []configuration.FieldOption{
	{Label: "Allow unencrypted and encrypted", Value: "ALLOW_UNENCRYPTED_AND_ENCRYPTED"},
	{Label: "Encrypted only", Value: "ENCRYPTED_ONLY"},
	{Label: "Trusted client certificate required", Value: "TRUSTED_CLIENT_CERTIFICATE_REQUIRED"},
}

var availabilityTypeOptions = []configuration.FieldOption{
	{Label: "Zonal (single zone)", Value: "ZONAL"},
	{Label: "Regional (high availability)", Value: "REGIONAL"},
}

var dataDiskTypeOptions = []configuration.FieldOption{
	{Label: "SSD", Value: "PD_SSD"},
	{Label: "HDD", Value: "PD_HDD"},
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
- **Region**: The region to create the instance in, chosen from the regions where Cloud SQL is available
- **Tier**: The machine tier (size), chosen from the predefined tiers available in the selected region. Custom machine types (` + "`db-custom-*`" + `) are not listed.
- **Disk Size (GB)**: The data disk size (minimum 10)
- **Disk Type**: SSD (default) or HDD
- **Edition**: Enterprise or Enterprise Plus
- **Availability**: Zonal (single zone, default) or Regional (adds a standby in another zone for high availability)
- **Root Password**: Initial password for the default admin user (optional, stored as a secret)
- **Assign Public IP**: Whether to give the instance a public IPv4 address (default yes)
- **SSL Mode**: How the instance enforces SSL/TLS on incoming connections (optional)
- **Authorized Networks**: CIDR ranges allowed to connect over the public IP (optional)
- **Automated Backups**: Enable daily automated backups (optional)
- **Deletion Protection**: Prevent the instance from being deleted until protection is removed (optional)
- **Labels**: Key-value labels applied to the instance (optional)

## Output

Emits a ` + "`gcp.cloudsql.instance`" + ` payload with the ready instance's ` + "`name`" + `, ` + "`state`" + `, ` + "`databaseVersion`" + `, ` + "`region`" + `, ` + "`tier`" + `, ` + "`connectionName`" + `, ` + "`ipAddress`" + `, and ` + "`selfLink`" + `.

## Important Notes

- **Instance creation is asynchronous and takes several minutes.** This component polls the instance until it reaches ` + "`RUNNABLE`" + ` (or times out) before emitting, so downstream steps run only once the instance is ready.
- A public IP with no **Authorized Networks** is reachable only through the Cloud SQL Auth Proxy or private access; add CIDR ranges to allow direct external clients.
- With **Deletion Protection** enabled, the Delete Instance component will fail until protection is removed.
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
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The region to create the instance in",
			Placeholder: "Select a region",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRegion,
				},
			},
		},
		{
			Name:        "tier",
			Label:       "Tier",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The machine tier (size). Select a region first; custom machine types are not listed.",
			Placeholder: "Select a tier",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTier,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
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
			Name:        "dataDiskType",
			Label:       "Disk Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "PD_SSD",
			Description: "The data disk type (SSD for most workloads, HDD for cheaper cold storage)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: dataDiskTypeOptions},
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
			Name:        "availabilityType",
			Label:       "Availability",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "ZONAL",
			Description: "Zonal runs in a single zone; Regional adds a standby in another zone for high availability",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: availabilityTypeOptions},
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
		{
			Name:        "publicIp",
			Label:       "Assign Public IP",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Assign a public IPv4 address to the instance. Disable only if you configure private connectivity.",
		},
		{
			Name:        "sslMode",
			Label:       "SSL Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "How the instance enforces SSL/TLS on incoming connections",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: sslModeOptions},
			},
		},
		{
			Name:        "authorizedNetworks",
			Label:       "Authorized Networks",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "CIDR ranges allowed to connect over the public IP (e.g. 203.0.113.0/24)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "CIDR",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "automatedBackups",
			Label:       "Automated Backups",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enable daily automated backups (Cloud SQL picks the backup window)",
		},
		{
			Name:        "deletionProtection",
			Label:       "Deletion Protection",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Prevent the instance from being deleted until protection is removed",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels for the instance (billing, environment, team)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key (e.g. env, team, cost-center)",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
	}
}

// labelsMap converts label entries to the userLabels map, skipping blank keys.
func labelsMap(entries []labelEntry) map[string]string {
	labels := make(map[string]string, len(entries))
	for _, e := range entries {
		key := strings.TrimSpace(e.Key)
		if key == "" {
			continue
		}
		labels[key] = strings.TrimSpace(e.Value)
	}
	return labels
}

// authorizedNetworkEntries maps CIDR strings to Cloud SQL ACL entries, skipping
// blanks.
func authorizedNetworkEntries(cidrs []string) []map[string]any {
	entries := make([]map[string]any, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		entries = append(entries, map[string]any{"value": c})
	}
	return entries
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
	if diskType := strings.TrimSpace(spec.DataDiskType); diskType != "" {
		settings["dataDiskType"] = diskType
	}
	if availability := strings.TrimSpace(spec.AvailabilityType); availability != "" {
		settings["availabilityType"] = availability
	}
	if spec.AutomatedBackups != nil && *spec.AutomatedBackups {
		settings["backupConfiguration"] = map[string]any{"enabled": true}
	}
	if labels := labelsMap(spec.Labels); len(labels) > 0 {
		settings["userLabels"] = labels
	}

	// IP / SSL connection security. Public IP defaults to on, matching Cloud SQL.
	ipConfig := map[string]any{
		"ipv4Enabled": spec.PublicIP == nil || *spec.PublicIP,
	}
	if sslMode := strings.TrimSpace(spec.SSLMode); sslMode != "" {
		ipConfig["sslMode"] = sslMode
	}
	if nets := authorizedNetworkEntries(spec.AuthorizedNetworks); len(nets) > 0 {
		ipConfig["authorizedNetworks"] = nets
	}
	settings["ipConfiguration"] = ipConfig

	if spec.DeletionProtection != nil {
		settings["deletionProtectionEnabled"] = *spec.DeletionProtection
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

	if _, err := createInstance(context.Background(), client, client.ProjectID(), body); err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to create instance", err, roleHintAdmin))
	}

	// Instance creation takes minutes, so record what to poll and schedule the
	// first poll instead of blocking this execution until the instance is ready.
	// Failures here are terminal: the GCP operation is already running, and a
	// plain error would roll back the request and re-run Execute against an
	// instance that already exists.
	if err := ctx.Metadata.Set(instanceExecMetadata{Instance: name}); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("instance creation started but failed to record poll state: %v", err))
	}
	if err := ctx.Requests.ScheduleActionCall(pollHookName, map[string]any{}, instancePollInterval); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("instance creation started but failed to schedule the status poll: %v", err))
	}
	return nil
}

// poll re-checks the instance until it reaches RUNNABLE (then emits it), fails,
// or the attempt/error budget is exhausted; otherwise it re-schedules itself.
// Terminal conditions fail the execution via ExecutionState.Fail — returning a
// plain error would roll back the request and leave the run in progress forever.
func (c *CreateInstance) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var md instanceExecMetadata
	if err := mapstructure.WeakDecode(ctx.Metadata.Get(), &md); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode poll metadata: %v", err))
	}
	if md.Instance == "" {
		return ctx.ExecutionState.Fail("error", "poll metadata is missing the instance name")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	inst, err := getInstance(context.Background(), client, client.ProjectID(), md.Instance)
	if err != nil {
		md.PollErrors++
		ctx.Logger.Warnf("failed to get instance %s (attempt %d/%d): %v", md.Instance, md.PollErrors, maxPollErrors, err)
		if md.PollErrors >= maxPollErrors {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("giving up polling instance %s after %d consecutive errors: %v", md.Instance, maxPollErrors, err))
		}
		if err := ctx.Metadata.Set(md); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(pollHookName, map[string]any{}, instancePollInterval)
	}

	md.PollErrors = 0
	md.PollAttempts++
	if err := ctx.Metadata.Set(md); err != nil {
		return err
	}

	switch inst.State {
	case instanceStateRunnable:
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, instancePayloadType, []any{instancePayload(inst)})
	case instanceStateFailed:
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("instance %s entered state %s", md.Instance, inst.State))
	default:
		if md.PollAttempts >= instanceMaxPollAttempts {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("timed out waiting for instance %s to reach RUNNABLE after %d polls (state: %s)", md.Instance, md.PollAttempts, inst.State))
		}
		return ctx.Requests.ScheduleActionCall(pollHookName, map[string]any{}, instancePollInterval)
	}
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
	return []core.Hook{{Name: pollHookName, Type: core.HookTypeInternal}}
}

func (c *CreateInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != pollHookName {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
	return c.poll(ctx)
}
