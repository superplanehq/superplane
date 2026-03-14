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

const (
	createRecordPayloadType   = "gcp.clouddns.change"
	createRecordOutputChannel = "default"
)

type CreateRecord struct{}

type CreateRecordConfiguration struct {
	ManagedZone string   `json:"managedZone" mapstructure:"managedZone"`
	Name        string   `json:"name" mapstructure:"name"`
	Type        string   `json:"type" mapstructure:"type"`
	TTL         int      `json:"ttl" mapstructure:"ttl"`
	Rrdatas     []string `json:"rrdatas" mapstructure:"rrdatas"`
}

func (c *CreateRecord) Name() string {
	return "gcp.clouddns.createRecord"
}

func (c *CreateRecord) Label() string {
	return "Cloud DNS • Create Record"
}

func (c *CreateRecord) Description() string {
	return "Create a DNS record in a Google Cloud DNS managed zone"
}

func (c *CreateRecord) Documentation() string {
	return `The Create Record component creates a new DNS record set in a Google Cloud DNS managed zone.

## Configuration

- **Managed Zone** (required): The Cloud DNS managed zone where the record will be created.
- **Record Name** (required): The DNS name for the record (e.g. ` + "`api.example.com`" + `). A trailing dot is added automatically.
- **Record Type** (required): The DNS record type (A, AAAA, CNAME, TXT, MX, etc.).
- **TTL** (required): Time to live in seconds. Defaults to 300.
- **Record Values** (required): The values for the record (e.g. IP addresses for A records).

## Required IAM roles

The service account must have ` + "`roles/dns.admin`" + ` or ` + "`roles/dns.editor`" + ` on the project.

## Output

- ` + "`change.id`" + `: The Cloud DNS change ID.
- ` + "`change.status`" + `: The change status (` + "`done`" + `).
- ` + "`change.startTime`" + `: When the change was submitted.
- ` + "`record.name`" + `: The DNS record name.
- ` + "`record.type`" + `: The DNS record type.`
}

func (c *CreateRecord) Icon() string  { return "gcp" }
func (c *CreateRecord) Color() string { return "gray" }

func (c *CreateRecord) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRecord) Configuration() []configuration.Field {
	fields := baseRecordConfigurationFields()
	fields = append(fields, ttlConfigurationField())
	fields = append(fields, rrdatasConfigurationField())
	return fields
}

func decodeCreateRecordConfig(raw any) (CreateRecordConfiguration, error) {
	var config CreateRecordConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return CreateRecordConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ManagedZone = strings.TrimSpace(config.ManagedZone)
	config.Name = normalizeRecordName(config.Name)
	config.Type = strings.TrimSpace(config.Type)
	config.Rrdatas = normalizeRrdatas(config.Rrdatas)
	return config, nil
}

func (c *CreateRecord) Setup(ctx core.SetupContext) error {
	config, err := decodeCreateRecordConfig(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateBaseConfig(config.ManagedZone, config.Name, config.Type); err != nil {
		return err
	}
	return validateRrdatas(config.Rrdatas)
}

func (c *CreateRecord) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateRecordConfig(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	ttl := config.TTL
	if ttl <= 0 {
		ttl = 300
	}

	record := ResourceRecordSet{
		Name:    config.Name,
		Type:    config.Type,
		TTL:     ttl,
		Rrdatas: config.Rrdatas,
	}

	change, err := applyChange(context.Background(), client, client.ProjectID(), config.ManagedZone, []ResourceRecordSet{record}, nil)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create DNS record: %v", err))
	}

	if change.Status == "done" {
		return ctx.ExecutionState.Emit(createRecordOutputChannel, createRecordPayloadType, []any{
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

func (c *CreateRecord) Actions() []core.Action {
	return []core.Action{
		{Name: pollChangeActionName, Description: "Poll for change status"},
	}
}

func (c *CreateRecord) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case pollChangeActionName:
		return pollChangeUntilDone(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *CreateRecord) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateRecord) Cancel(_ core.ExecutionContext) error { return nil }
func (c *CreateRecord) Cleanup(_ core.SetupContext) error    { return nil }
func (c *CreateRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func buildChangeOutput(change *ChangeInfo, recordName, recordType string) map[string]any {
	return map[string]any{
		"change": map[string]any{
			"id":        change.ID,
			"status":    change.Status,
			"startTime": change.StartTime,
		},
		"record": map[string]any{
			"name": recordName,
			"type": recordType,
		},
	}
}
