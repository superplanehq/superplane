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

type DeleteRecord struct{}

type DeleteRecordConfiguration struct {
	ManagedZone string `json:"managedZone" mapstructure:"managedZone"`
	Name        string `json:"name" mapstructure:"name"`
	Type        string `json:"type" mapstructure:"type"`
}

func (c *DeleteRecord) Name() string {
	return "gcp.clouddns.deleteRecord"
}

func (c *DeleteRecord) Label() string {
	return "Cloud DNS • Delete Record"
}

func (c *DeleteRecord) Description() string {
	return "Delete a DNS record from a Google Cloud DNS managed zone"
}

func (c *DeleteRecord) Documentation() string {
	return `The Delete Record component deletes a DNS record set from a Google Cloud DNS managed zone.

## Configuration

- **Managed Zone** (required): The Cloud DNS managed zone containing the record.
- **Record Name** (required): The DNS name of the record to delete (e.g. ` + "`api.example.com`" + `).
- **Record Type** (required): The DNS record type (A, AAAA, CNAME, TXT, MX, etc.).

## Required IAM roles

The service account must have ` + "`roles/dns.admin`" + ` or ` + "`roles/dns.editor`" + ` on the project.

## Output

- ` + "`change.id`" + `: The Cloud DNS change ID.
- ` + "`change.status`" + `: The change status (` + "`done`" + `).
- ` + "`change.startTime`" + `: When the change was submitted.
- ` + "`record.name`" + `: The DNS record name.
- ` + "`record.type`" + `: The DNS record type.`
}

func (c *DeleteRecord) Icon() string  { return "gcp" }
func (c *DeleteRecord) Color() string { return "gray" }

func (c *DeleteRecord) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteRecord) Configuration() []configuration.Field {
	return baseRecordConfigurationFields()
}

func decodeDeleteRecordConfig(raw any) (DeleteRecordConfiguration, error) {
	var config DeleteRecordConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return DeleteRecordConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ManagedZone = strings.TrimSpace(config.ManagedZone)
	config.Name = normalizeRecordName(config.Name)
	config.Type = strings.TrimSpace(config.Type)
	return config, nil
}

func (c *DeleteRecord) Setup(ctx core.SetupContext) error {
	config, err := decodeDeleteRecordConfig(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateBaseConfig(config.ManagedZone, config.Name, config.Type)
}

func (c *DeleteRecord) Execute(ctx core.ExecutionContext) error {
	config, err := decodeDeleteRecordConfig(ctx.Configuration)
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

	change, err := applyChange(context.Background(), client, projectID, config.ManagedZone, nil, []ResourceRecordSet{*existing})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete DNS record: %v", err))
	}

	if change.Status == "done" {
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.clouddns.change", []any{
			buildChangeOutput(change, config.Name, config.Type),
		})
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

func (c *DeleteRecord) Actions() []core.Action {
	return []core.Action{
		{Name: pollChangeActionName, Description: "Poll for change status"},
	}
}

func (c *DeleteRecord) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case pollChangeActionName:
		return pollChangeUntilDone(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *DeleteRecord) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteRecord) Cancel(_ core.ExecutionContext) error { return nil }
func (c *DeleteRecord) Cleanup(_ core.SetupContext) error    { return nil }
func (c *DeleteRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
