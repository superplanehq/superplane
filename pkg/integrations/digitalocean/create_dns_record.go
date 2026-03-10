package digitalocean

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateDNSRecord struct{}

type CreateDNSRecordSpec struct {
	Domain     string `json:"domain" mapstructure:"domain"`
	RecordType string `json:"recordType" mapstructure:"recordType"`
	Name       string `json:"name" mapstructure:"name"`
	Data       string `json:"data" mapstructure:"data"`
	TTL        int    `json:"ttl" mapstructure:"ttl"`
}

func (c *CreateDNSRecord) Name() string {
	return "digitalocean.createDNSRecord"
}

func (c *CreateDNSRecord) Label() string {
	return "Create DNS Record"
}

func (c *CreateDNSRecord) Description() string {
	return "Create a DNS record for a DigitalOcean domain"
}

func (c *CreateDNSRecord) Documentation() string {
	return `The Create DNS Record component creates a new DNS record in a DigitalOcean domain.

## Use Cases

- **DNS management**: Automate DNS record creation as part of infrastructure provisioning
- **Blue/green deployments**: Create DNS records pointing to new infrastructure
- **Certificate validation**: Create DNS records for domain verification

## Configuration

- **Domain**: The domain to create the record for (required, supports expressions)
- **Record Type**: The DNS record type (required). One of: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
- **Name**: The hostname for the record (required, supports expressions). Use @ for apex domain.
- **Data**: The record data/value (required, supports expressions)
- **TTL**: Time-to-live in seconds (optional, defaults to 1800)

## Output

Returns the created DNS record including:
- **id**: Record ID
- **type**: Record type
- **name**: Record name
- **data**: Record data
- **ttl**: Record TTL`
}

func (c *CreateDNSRecord) Icon() string {
	return "server"
}

func (c *CreateDNSRecord) Color() string {
	return "gray"
}

func (c *CreateDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"id":   12345,
			"type": "A",
			"name": "www",
			"data": "104.131.186.241",
			"ttl":  1800,
		},
	}
}

func (c *CreateDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The domain to create the record for",
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

func (c *CreateDNSRecord) Setup(ctx core.SetupContext) error {
	spec := CreateDNSRecordSpec{}
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

func (c *CreateDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := CreateDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	domain := readStringFromAny(spec.Domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	ttl := spec.TTL
	if ttl == 0 {
		ttl = 1800
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	record, err := client.CreateDNSRecord(domain, CreateDNSRecordRequest{
		Type: spec.RecordType,
		Name: readStringFromAny(spec.Name),
		Data: readStringFromAny(spec.Data),
		TTL:  ttl,
	})
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.dns_record.created",
		[]any{record},
	)
}

func (c *CreateDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
