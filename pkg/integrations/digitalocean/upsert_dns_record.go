package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpsertDNSRecord struct{}

type UpsertDNSRecordSpec struct {
	Domain   string `json:"domain"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	TTL      string `json:"ttl"`
	Priority string `json:"priority"`
	Port     string `json:"port"`
	Weight   string `json:"weight"`
}

func (u *UpsertDNSRecord) Name() string {
	return "digitalocean.upsertDNSRecord"
}

func (u *UpsertDNSRecord) Label() string {
	return "Upsert DNS Record"
}

func (u *UpsertDNSRecord) Description() string {
	return "Idempotently create or update a DNS record for a DigitalOcean domain"
}

func (u *UpsertDNSRecord) Documentation() string {
	return `The Upsert DNS Record component idempotently creates or updates a DNS record for a DigitalOcean-managed domain.

It first looks up existing records with the same name and type. If a match is found it updates the record in-place; otherwise it creates a new one.

## Use Cases

- **Idempotent provisioning**: Safely run DNS setup steps multiple times without creating duplicates
- **IP updates**: Keep A/AAAA records in sync with changing IP addresses
- **Dynamic configuration**: Update TXT records (e.g. SPF, DKIM) as part of automated workflows

## Configuration

- **Domain**: The DigitalOcean-managed domain to manage the record in (required)
- **Type**: The DNS record type (required): A, AAAA, CNAME, MX, NS, TXT, SRV, CAA
- **Name**: The subdomain name for the record (required, use @ for root)
- **Data**: The record value, e.g. an IP address or hostname (required, supports expressions)
- **TTL**: Time-to-live in seconds (optional, defaults to 1800)
- **Priority**: Record priority for MX/SRV records (optional)
- **Port**: Port number for SRV records (optional)
- **Weight**: Weight for SRV records (optional)

## Output

Returns the created or updated DNS record including:
- **id**: Record ID
- **type**: Record type
- **name**: Subdomain name
- **data**: Record value
- **ttl**: Time-to-live
- **priority**: Priority (for MX/SRV)
- **port**: Port (for SRV)
- **weight**: Weight (for SRV)`
}

func (u *UpsertDNSRecord) Icon() string {
	return "refresh-cw"
}

func (u *UpsertDNSRecord) Color() string {
	return "teal"
}

func (u *UpsertDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpsertDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The DigitalOcean-managed domain to manage the record in",
			Placeholder: "Select domain",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "domain",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "type",
			Label:    "Record Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "A",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: dnsRecordTypeOptions,
				},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The subdomain name (use @ for the root domain)",
			Placeholder: "www",
		},
		{
			Name:        "data",
			Label:       "Data",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The record value (e.g. IP address, hostname, or text)",
		},
		{
			Name:        "ttl",
			Label:       "TTL (seconds)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Default:     "1800",
			Description: "Time-to-live in seconds",
			Placeholder: "1800",
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Record priority (required for MX and SRV records)",
			Placeholder: "10",
		},
		{
			Name:        "port",
			Label:       "Port",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Port number (required for SRV records)",
			Placeholder: "443",
		},
		{
			Name:        "weight",
			Label:       "Weight",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Weight (required for SRV records)",
			Placeholder: "10",
		},
	}
}

func (u *UpsertDNSRecord) Setup(ctx core.SetupContext) error {
	spec := UpsertDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Domain == "" {
		return errors.New("domain is required")
	}

	if spec.Type == "" {
		return errors.New("type is required")
	}

	if !isValidDNSRecordType(spec.Type) {
		return fmt.Errorf("invalid record type %q", spec.Type)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Data == "" {
		return errors.New("data is required")
	}

	return nil
}

func (u *UpsertDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := UpsertDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	ttl := 1800
	if spec.TTL != "" {
		parsed, err := strconv.Atoi(spec.TTL)
		if err != nil {
			return fmt.Errorf("invalid ttl value %q: %v", spec.TTL, err)
		}
		ttl = parsed
	}

	var priority *int
	if spec.Priority != "" {
		p, err := strconv.Atoi(spec.Priority)
		if err != nil {
			return fmt.Errorf("invalid priority value %q: %v", spec.Priority, err)
		}
		priority = &p
	}

	var port *int
	if spec.Port != "" {
		p, err := strconv.Atoi(spec.Port)
		if err != nil {
			return fmt.Errorf("invalid port value %q: %v", spec.Port, err)
		}
		port = &p
	}

	var weight *int
	if spec.Weight != "" {
		w, err := strconv.Atoi(spec.Weight)
		if err != nil {
			return fmt.Errorf("invalid weight value %q: %v", spec.Weight, err)
		}
		weight = &w
	}

	req := DNSRecordRequest{
		Type:     spec.Type,
		Name:     spec.Name,
		Data:     spec.Data,
		TTL:      ttl,
		Priority: priority,
		Port:     port,
		Weight:   weight,
	}

	// Look for an existing record with the same name and type
	existing, err := client.ListDNSRecords(spec.Domain)
	if err != nil {
		return fmt.Errorf("failed to list DNS records: %v", err)
	}

	for _, record := range existing {
		if record.Type == spec.Type && record.Name == spec.Name {
			// Found a match — update in place
			updated, err := client.UpdateDNSRecord(spec.Domain, record.ID, req)
			if err != nil {
				return fmt.Errorf("failed to update DNS record: %v", err)
			}

			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.dns.record.upserted",
				[]any{updated},
			)
		}
	}

	// No match found — create a new record
	created, err := client.CreateDNSRecord(spec.Domain, req)
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.dns.record.upserted",
		[]any{created},
	)
}

func (u *UpsertDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpsertDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpsertDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (u *UpsertDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (u *UpsertDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpsertDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
