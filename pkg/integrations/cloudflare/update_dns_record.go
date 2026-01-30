package cloudflare

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	DNSRecordPayloadType          = "cloudflare.dnsRecord"
	DNSRecordSuccessOutputChannel = "success"
	DNSRecordFailedOutputChannel  = "failed"
)

type UpdateDNSRecord struct{}

type UpdateDNSRecordSpec struct {
	Zone     string `json:"zone"`
	RecordID string `json:"recordId"`

	Content *string `json:"content,omitempty"`
	TTL     *int    `json:"ttl,omitempty"`
	Proxied *bool   `json:"proxied,omitempty"`
	Name    *string `json:"name,omitempty"`
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
- **Record ID**: DNS record ID to update
- **Content**: New record value (optional)
- **TTL**: New TTL in seconds or 1 for auto (optional)
- **Proxied**: Proxy through Cloudflare (optional)
- **Name**: New record name (optional)

## Output Channels

- **Success**: Emits the updated DNS record (id, type, name, content, proxied, ttl)
- **Failed**: Emits an error payload when the record cannot be updated (not found, invalid update, etc.)`
}

func (c *UpdateDNSRecord) Icon() string {
	return "cloud"
}

func (c *UpdateDNSRecord) Color() string {
	return "orange"
}

func (c *UpdateDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  DNSRecordSuccessOutputChannel,
			Label: "Success",
		},
		{
			Name:  DNSRecordFailedOutputChannel,
			Label: "Failed",
		},
	}
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
			Label:       "Record ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the DNS record to update",
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New record value (e.g. IP address for A record, hostname for CNAME)",
			Placeholder: "1.2.3.4",
		},
		{
			Name:        "ttl",
			Label:       "TTL",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "TTL in seconds, or 1 for auto",
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
			Required:    false,
			Togglable:   true,
			Description: "Whether Cloudflare should proxy traffic for this record",
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New record name (e.g. subdomain or full record name)",
			Placeholder: "www.example.com",
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

	if spec.Content != nil && *spec.Content == "" {
		return errors.New("content cannot be empty when provided")
	}

	if spec.Name != nil && *spec.Name == "" {
		return errors.New("name cannot be empty when provided")
	}

	if spec.TTL != nil && *spec.TTL < 1 {
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

	zoneID := c.resolveZoneID(spec.Zone, ctx.Integration.GetMetadata())

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	existing, err := client.GetDNSRecord(zoneID, spec.RecordID)
	if err != nil {
		return ctx.ExecutionState.Emit(DNSRecordFailedOutputChannel, DNSRecordPayloadType, []any{
			map[string]any{
				"error":    err.Error(),
				"zoneId":   zoneID,
				"recordId": spec.RecordID,
			},
		})
	}

	updateReq := UpdateDNSRecordRequest{
		Type:    existing.Type,
		Name:    existing.Name,
		Content: existing.Content,
		TTL:     existing.TTL,
		Proxied: existing.Proxied,
	}

	if spec.Content != nil {
		updateReq.Content = *spec.Content
	}
	if spec.TTL != nil {
		updateReq.TTL = *spec.TTL
	}
	if spec.Proxied != nil {
		updateReq.Proxied = *spec.Proxied
	}
	if spec.Name != nil {
		updateReq.Name = *spec.Name
	}

	updated, err := client.UpdateDNSRecord(zoneID, spec.RecordID, updateReq)
	if err != nil {
		return ctx.ExecutionState.Emit(DNSRecordFailedOutputChannel, DNSRecordPayloadType, []any{
			map[string]any{
				"error":    err.Error(),
				"zoneId":   zoneID,
				"recordId": spec.RecordID,
			},
		})
	}

	return ctx.ExecutionState.Emit(DNSRecordSuccessOutputChannel, DNSRecordPayloadType, []any{
		map[string]any{
			"zoneId":   zoneID,
			"recordId": updated.ID,
			"record":   updated,
		},
	})
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

