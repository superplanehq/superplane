package clouddns

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

type UpdateRecord struct{}

type UpdateRecordConfiguration struct {
	ManagedZone string   `json:"managedZone" mapstructure:"managedZone"`
	Name        string   `json:"name" mapstructure:"name"`
	Type        string   `json:"type" mapstructure:"type"`
	TTL         int      `json:"ttl" mapstructure:"ttl"`
	Rrdatas     []string `json:"rrdatas" mapstructure:"rrdatas"`
}

func (c *UpdateRecord) Name() string {
	return "gcp.clouddns.updateRecord"
}

func (c *UpdateRecord) Label() string {
	return "Cloud DNS • Update Record"
}

func (c *UpdateRecord) Description() string {
	return "Update an existing DNS record in a Google Cloud DNS managed zone"
}

func (c *UpdateRecord) Documentation() string {
	return `The Update Record component updates an existing DNS record set in a Google Cloud DNS managed zone.

## Configuration

- **Managed Zone** (required): The Cloud DNS managed zone containing the record.
- **Record Name** (required): The DNS name of the record to update (e.g. ` + "`api.example.com`" + `).
- **Record Type** (required): The DNS record type (A, AAAA, CNAME, TXT, MX, etc.).
- **TTL** (required): New time to live in seconds.
- **Record Values** (required): The new values for the record.

## Required IAM roles

The service account must have ` + "`roles/dns.admin`" + ` or ` + "`roles/dns.editor`" + ` on the project.

## Output

- ` + "`change.id`" + `: The Cloud DNS change ID.
- ` + "`change.status`" + `: The change status (` + "`done`" + `).
- ` + "`change.startTime`" + `: When the change was submitted.
- ` + "`record.name`" + `: The DNS record name.
- ` + "`record.type`" + `: The DNS record type.`
}

func (c *UpdateRecord) Icon() string  { return "gcp" }
func (c *UpdateRecord) Color() string { return "gray" }

func (c *UpdateRecord) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateRecord) Configuration() []configuration.Field {
	fields := baseRecordConfigurationFields()
	fields = append(fields, ttlConfigurationField())
	fields = append(fields, rrdatasConfigurationField())
	return fields
}

func decodeUpdateRecordConfig(raw any) (UpdateRecordConfiguration, error) {
	var config UpdateRecordConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return UpdateRecordConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ManagedZone = strings.TrimSpace(config.ManagedZone)
	config.Name = normalizeRecordName(config.Name)
	config.Type = strings.TrimSpace(config.Type)
	config.Rrdatas = normalizeRrdatas(config.Rrdatas)
	return config, nil
}

func (c *UpdateRecord) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdateRecordConfig(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateBaseConfig(config.ManagedZone, config.Name, config.Type); err != nil {
		return err
	}
	return validateRrdatas(config.Rrdatas)
}

func (c *UpdateRecord) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdateRecordConfig(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	existing, err := getRecordSet(context.Background(), client, projectID, config.ManagedZone, config.Name, config.Type)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to look up existing record: %v", err))
	}
	if existing == nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("record %s %s not found in zone %s", config.Name, config.Type, config.ManagedZone))
	}

	ttl := config.TTL
	if ttl <= 0 {
		ttl = existing.TTL
	}

	updated := ResourceRecordSet{
		Name:    config.Name,
		Type:    config.Type,
		TTL:     ttl,
		Rrdatas: config.Rrdatas,
	}

	change, err := applyChange(context.Background(), client, projectID, config.ManagedZone, []ResourceRecordSet{updated}, []ResourceRecordSet{*existing})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update DNS record: %v", err))
	}

	if change.Status == "done" {
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.clouddns.change", []any{
			buildChangeOutput(change, config.Name, config.Type),
		})
	}

	if change.Status != "pending" {
		return ctx.ExecutionState.Fail(
			"error",
			fmt.Sprintf("unexpected Cloud DNS change status %q for change %q", change.Status, change.ID),
		)
	}

	if err := ctx.Metadata.Set(RecordSetPollMetadata{
		ChangeID:    change.ID,
		ManagedZone: config.ManagedZone,
		RecordName:  config.Name,
		RecordType:  config.Type,
		StartTime:   change.StartTime,
	}); err != nil {
		return fmt.Errorf("failed to set poll metadata: %w", err)
	}
	return ctx.Requests.ScheduleActionCall(pollChangeActionName, map[string]any{}, pollInterval)
}

func (c *UpdateRecord) Actions() []core.Action {
	return []core.Action{
		{Name: pollChangeActionName, Description: "Poll for change status"},
	}
}

func (c *UpdateRecord) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case pollChangeActionName:
		return pollChangeUntilDone(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *UpdateRecord) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateRecord) Cancel(_ core.ExecutionContext) error { return nil }
func (c *UpdateRecord) Cleanup(_ core.SetupContext) error    { return nil }
func (c *UpdateRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
