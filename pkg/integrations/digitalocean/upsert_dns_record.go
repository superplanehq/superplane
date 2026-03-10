package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpsertDNSRecord struct{}

type UpsertDNSRecordSpec struct {
	Domain     string `json:"domain" mapstructure:"domain"`
	RecordType string `json:"recordType" mapstructure:"recordType"`
	Name       string `json:"name" mapstructure:"name"`
	Data       string `json:"data" mapstructure:"data"`
	TTL        int    `json:"ttl" mapstructure:"ttl"`
}

func (c *UpsertDNSRecord) Name() string {
	return "digitalocean.upsertDNSRecord"
}

func (c *UpsertDNSRecord) Label() string {
	return "Upsert DNS Record"
}

func (c *UpsertDNSRecord) Description() string {
	return "Create or update a DNS record for a DigitalOcean domain"
}

func (c *UpsertDNSRecord) Documentation() string {
	return `The Upsert DNS Record component provides an idempotent create-or-update flow for DNS records.

## How It Works

1. Lists existing DNS records for the domain
2. Looks for a record matching the same type and name
3. If found, updates the existing record with new data
4. If not found, creates a new record
5. Emits the resulting record on the default output

## Configuration

- **Domain**: The domain to manage the record for (required, supports expressions)
- **Record Type**: The DNS record type (required). One of: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
- **Name**: The hostname for the record (required, supports expressions). Use @ for apex domain.
- **Data**: The record data/value (required, supports expressions)
- **TTL**: Time-to-live in seconds (optional, defaults to 1800)

## Output

Returns the created or updated DNS record including:
- **id**: Record ID
- **type**: Record type
- **name**: Record name
- **data**: Record data
- **ttl**: Record TTL
- **action**: Whether the record was "created" or "updated"`
}

func (c *UpsertDNSRecord) Icon() string {
	return "server"
}

func (c *UpsertDNSRecord) Color() string {
	return "gray"
}

func (c *UpsertDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpsertDNSRecord) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"id":     12345,
			"type":   "A",
			"name":   "www",
			"data":   "104.131.186.241",
			"ttl":    1800,
			"action": "created",
		},
	}
}

func (c *UpsertDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The domain to manage the record for",
		},
		{
			Name:        "recordType",
			Label:       "Record Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The DNS record type",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "A", Value: "A"},
						{Label: "AAAA", Value: "AAAA"},
						{Label: "CNAME", Value: "CNAME"},
						{Label: "MX", Value: "MX"},
						{Label: "TXT", Value: "TXT"},
						{Label: "NS", Value: "NS"},
						{Label: "SRV", Value: "SRV"},
						{Label: "CAA", Value: "CAA"},
					},
				},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The hostname for the record (use @ for apex domain)",
		},
		{
			Name:        "data",
			Label:       "Data",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The record data/value",
		},
		{
			Name:        "ttl",
			Label:       "TTL",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Time-to-live in seconds (defaults to 1800)",
		},
	}
}

func (c *UpsertDNSRecord) Setup(ctx core.SetupContext) error {
	spec := UpsertDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	if spec.RecordType == "" {
		return fmt.Errorf("recordType is required")
	}

	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}

	if spec.Data == "" {
		return fmt.Errorf("data is required")
	}

	return nil
}

func (c *UpsertDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := UpsertDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	domain := readStringFromAny(spec.Domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	recordName := readStringFromAny(spec.Name)
	recordData := readStringFromAny(spec.Data)
	ttl := spec.TTL
	if ttl == 0 {
		ttl = 1800
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	records, err := client.ListDNSRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to list DNS records: %v", err)
	}

	var existingRecord *DNSRecord
	for i := range records {
		if records[i].Type == spec.RecordType && records[i].Name == recordName {
			existingRecord = &records[i]
			break
		}
	}

	if existingRecord != nil {
		record, err := client.UpdateDNSRecord(domain, existingRecord.ID, UpdateDNSRecordRequest{
			Data: recordData,
			TTL:  ttl,
		})
		if err != nil {
			return fmt.Errorf("failed to update DNS record: %v", err)
		}

		payload := map[string]any{
			"id":     record.ID,
			"type":   record.Type,
			"name":   record.Name,
			"data":   record.Data,
			"ttl":    record.TTL,
			"action": "updated",
		}
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.dns_record.upserted",
			[]any{payload},
		)
	}

	record, err := client.CreateDNSRecord(domain, CreateDNSRecordRequest{
		Type: spec.RecordType,
		Name: recordName,
		Data: recordData,
		TTL:  ttl,
	})
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %v", err)
	}

	payload := map[string]any{
		"id":     record.ID,
		"type":   record.Type,
		"name":   record.Name,
		"data":   record.Data,
		"ttl":    record.TTL,
		"action": "created",
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.dns_record.upserted",
		[]any{payload},
	)
}

func (c *UpsertDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpsertDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpsertDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpsertDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpsertDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *UpsertDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
