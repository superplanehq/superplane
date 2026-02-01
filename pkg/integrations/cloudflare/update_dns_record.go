package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DNSRecordPayloadType = "cloudflare.dnsRecord"

type UpdateDNSRecord struct{}

type UpdateDNSRecordSpec struct {
	Zone     string `json:"zone"`
	RecordID string `json:"recordId"`

	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

func (c *UpdateDNSRecord) Name() string {
	return "cloudflare.updateDNSRecord"
}

func (c *UpdateDNSRecord) Label() string {
	return "Update DNS Record"
}

func (c *UpdateDNSRecord) Description() string {
	return "Update an existing DNS record in a Cloudflare zone"
}

func (c *UpdateDNSRecord) Documentation() string {
	return `The Update DNS Record component updates an existing DNS record in a Cloudflare zone.

## Use Cases

- **Infrastructure changes**: Update record content when an IP or target changes
- **Release automation**: Switch a record to proxied or adjust TTL during a migration
- **Verification**: Update TXT records for ownership verification as part of workflows

## Configuration

- **Zone**: Zone ID or domain name (recommended: select a zone from the integration)
- **Record**: DNS record to update (select from the list)
- **Content**: New record value
- **TTL**: TTL in seconds (default 360)
- **Proxied**: Whether Cloudflare should proxy traffic for this record

## Output

Emits the updated DNS record (id, type, name, content, proxied, ttl) on the default channel. If the update fails (e.g. record not found, invalid update), the component goes to an error state and does not emit.`
}

func (c *UpdateDNSRecord) Icon() string {
	return "cloud"
}

func (c *UpdateDNSRecord) Color() string {
	return "orange"
}

func (c *UpdateDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare zone containing the DNS record",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		},
		{
			Name:        "recordId",
			Label:       "Record",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The DNS record to update",
			Placeholder: "Select a record",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "dns_record",
				},
			},
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "New record value (e.g. IP address for A record, hostname for CNAME)",
			Placeholder: "1.2.3.4",
		},
		{
			Name:        "ttl",
			Label:       "TTL",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     360,
			Description: "TTL in seconds",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
				},
			},
		},
		{
			Name:        "proxied",
			Label:       "Proxied",
			Type:        configuration.FieldTypeBool,
			Required:    true,
			Default:     false,
			Description: "Whether Cloudflare should proxy traffic for this record",
		},
	}
}

func (c *UpdateDNSRecord) Setup(ctx core.SetupContext) error {
	spec := UpdateDNSRecordSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Zone == "" {
		return errors.New("zone is required")
	}

	if spec.RecordID == "" {
		return errors.New("recordId is required")
	}

	if spec.Content == "" {
		return errors.New("content is required")
	}

	if spec.TTL < 1 {
		return errors.New("ttl must be >= 1")
	}

	return nil
}

func (c *UpdateDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := UpdateDNSRecordSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	zoneID, recordID := c.parseZoneAndRecordID(spec.RecordID, spec.Zone, ctx.Integration.GetMetadata())

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	existing, err := client.GetDNSRecord(zoneID, recordID)
	if err != nil {
		return fmt.Errorf("get DNS record: %w", err)
	}

	updateReq := UpdateDNSRecordRequest{
		Type:    existing.Type,
		Name:    existing.Name,
		Content: existing.Content,
		TTL:     existing.TTL,
		Proxied: existing.Proxied,
	}

	updateReq.Content = spec.Content
	updateReq.TTL = spec.TTL
	updateReq.Proxied = spec.Proxied

	updated, err := client.UpdateDNSRecord(zoneID, recordID, updateReq)
	if err != nil {
		return fmt.Errorf("update DNS record: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DNSRecordPayloadType, []any{
		map[string]any{
			"zoneId":   zoneID,
			"recordId": updated.ID,
			"record":   updated,
		},
	})
}

// parseZoneAndRecordID returns zoneID and recordID. If recordIDValue contains "/" (from
// integration resource selection), it is split into zone and record ID; otherwise zone
// is resolved from zoneValue and recordIDValue is used as the record ID (backward compat).
func (c *UpdateDNSRecord) parseZoneAndRecordID(recordIDValue, zoneValue string, integrationMetadata any) (zoneID, recordID string) {
	if zonePart, recordPart, ok := strings.Cut(recordIDValue, "/"); ok && zonePart != "" && recordPart != "" {
		return zonePart, recordPart
	}
	return c.resolveZoneID(zoneValue, integrationMetadata), recordIDValue
}

func (c *UpdateDNSRecord) resolveZoneID(zone string, integrationMetadata any) string {
	metadata := Metadata{}
	if err := mapstructure.Decode(integrationMetadata, &metadata); err != nil {
		return zone
	}

	for _, z := range metadata.Zones {
		if z.ID == zone || z.Name == zone {
			return z.ID
		}
	}

	return zone
}

func (c *UpdateDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
