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

type CreateDNSRecord struct{}

type CreateDNSRecordSpec struct {
	Domain   string `json:"domain"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	TTL      string `json:"ttl"`
	Priority string `json:"priority"`
	Port     string `json:"port"`
	Weight   string `json:"weight"`
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
	return `The Create DNS Record component creates a new DNS record for a domain managed by DigitalOcean.

## Use Cases

- **Service discovery**: Add A or CNAME records when provisioning new services
- **Email routing**: Create MX records for custom mail delivery
- **Verification**: Add TXT records for domain ownership verification
- **Subdomain management**: Dynamically create subdomains as part of provisioning workflows

## Configuration

- **Domain**: The DigitalOcean-managed domain to add the record to (required)
- **Type**: The DNS record type (required): A, AAAA, CNAME, MX, NS, TXT, SRV, CAA
- **Name**: The subdomain name for the record (required, use @ for root)
- **Data**: The record value, e.g. an IP address or hostname (required, supports expressions)
- **TTL**: Time-to-live in seconds (optional, defaults to 1800)
- **Priority**: Record priority for MX/SRV records (optional)
- **Port**: Port number for SRV records (optional)
- **Weight**: Weight for SRV records (optional)

## Output

Returns the created DNS record including:
- **id**: Record ID
- **type**: Record type
- **name**: Subdomain name
- **data**: Record value
- **ttl**: Time-to-live
- **priority**: Priority (for MX/SRV)
- **port**: Port (for SRV)
- **weight**: Weight (for SRV)`
}

func (c *CreateDNSRecord) Icon() string {
	return "globe"
}

func (c *CreateDNSRecord) Color() string {
	return "blue"
}

func (c *CreateDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDNSRecord) Configuration() []configuration.Field {
	return dnsRecordConfiguration()
}

func (c *CreateDNSRecord) Setup(ctx core.SetupContext) error {
	spec := CreateDNSRecordSpec{}
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

func (c *CreateDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := CreateDNSRecordSpec{}
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

	record, err := client.CreateDNSRecord(spec.Domain, DNSRecordRequest{
		Type:     spec.Type,
		Name:     spec.Name,
		Data:     spec.Data,
		TTL:      ttl,
		Priority: priority,
		Port:     port,
		Weight:   weight,
	})
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.dns.record.created",
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

func (c *CreateDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
