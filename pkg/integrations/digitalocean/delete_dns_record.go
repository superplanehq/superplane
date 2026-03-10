package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteDNSRecord struct{}

type DeleteDNSRecordSpec struct {
	Domain   string `json:"domain" mapstructure:"domain"`
	RecordID string `json:"recordId" mapstructure:"recordId"`
}

func (c *DeleteDNSRecord) Name() string {
	return "digitalocean.deleteDNSRecord"
}

func (c *DeleteDNSRecord) Label() string {
	return "Delete DNS Record"
}

func (c *DeleteDNSRecord) Description() string {
	return "Delete a DNS record from a DigitalOcean domain"
}

func (c *DeleteDNSRecord) Documentation() string {
	return `The Delete DNS Record component deletes a DNS record from a DigitalOcean domain.

## How It Works

1. Deletes the specified DNS record via the DigitalOcean API
2. Emits on the default output when the record is deleted. If deletion fails, the execution errors.

## Configuration

- **Domain**: The domain the record belongs to (required, supports expressions)
- **Record ID**: The ID of the DNS record to delete (required, supports expressions)

## Output

Returns confirmation of the deleted record:
- **domain**: The domain name
- **recordId**: The ID of the deleted record`
}

func (c *DeleteDNSRecord) Icon() string {
	return "server"
}

func (c *DeleteDNSRecord) Color() string {
	return "gray"
}

func (c *DeleteDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteDNSRecord) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"domain":   "example.com",
			"recordId": 12345,
		},
	}
}

func (c *DeleteDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The domain the record belongs to",
		},
		{
			Name:        "recordId",
			Label:       "Record ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the DNS record to delete",
		},
	}
}

func (c *DeleteDNSRecord) Setup(ctx core.SetupContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	if spec.RecordID == "" {
		return fmt.Errorf("recordId is required")
	}

	return nil
}

func (c *DeleteDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	domain := readStringFromAny(spec.Domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	recordID, err := resolveIntID(ctx.Configuration, "recordId")
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(map[string]any{"domain": domain, "recordId": recordID}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteDNSRecord(domain, recordID); err != nil {
		return fmt.Errorf("failed to delete DNS record: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.dns_record.deleted",
		[]any{map[string]any{"domain": domain, "recordId": recordID}},
	)
}

func (c *DeleteDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *DeleteDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
