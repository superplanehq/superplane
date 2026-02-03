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

type DeleteDNSRecord struct{}

type DeleteDNSRecordSpec struct {
	Record string `json:"record"`
}

func (c *DeleteDNSRecord) Name() string {
	return "cloudflare.deleteDnsRecord"
}

func (c *DeleteDNSRecord) Label() string {
	return "Delete DNS Record"
}

func (c *DeleteDNSRecord) Description() string {
	return "Delete a DNS record from a Cloudflare zone"
}

func (c *DeleteDNSRecord) Documentation() string {
	return `The Delete DNS Record component removes a DNS record from a Cloudflare zone.

## Use Cases

- **Deprovisioning**: Remove DNS records when services or environments are torn down
- **Cleanup**: Delete temporary verification records (e.g. migration or certificate validation)
- **Maintenance**: Remove stale or incorrect records as part of workflow automation

## Configuration

- **Record**: Select the DNS record to delete (e.g. zone-id/record-id or record name)

## Output

Emits the deleted DNS record (zoneId, recordId, record) on the default channel. If the record is not found or deletion fails, the component goes to an error state and does not emit.`
}

func (c *DeleteDNSRecord) Icon() string {
	return "cloud"
}

func (c *DeleteDNSRecord) Color() string {
	return "orange"
}

func (c *DeleteDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "record",
			Label:       "Record",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The DNS record to delete",
			Placeholder: "{{ $['cloudflare.createDnsRecord'].data.id }}",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "dns_record",
				},
			},
		},
	}
}

func (c *DeleteDNSRecord) Setup(ctx core.SetupContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Record) == "" {
		return errors.New("record is required")
	}

	return nil
}

func (c *DeleteDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var zoneID, recordID string
	if strings.Contains(spec.Record, "/") {
		zoneFromConfig := ""
		if m, ok := ctx.Configuration.(map[string]any); ok {
			if z, ok := m["zone"]; ok && z != nil {
				zoneFromConfig = fmt.Sprint(z)
			}
		}
		zoneID, recordID = c.parseZoneAndRecordID(spec.Record, zoneFromConfig, ctx.Integration.GetMetadata())
	} else {
		recordValue := strings.TrimSpace(spec.Record)
		nameErr := error(nil)
		zoneID, recordID, nameErr = c.resolveRecordName(recordValue, client, ctx.Integration.GetMetadata())
		if nameErr != nil {
			// Fall back to resolving by record ID (e.g. from createDnsRecord.data.id)
			idErr := error(nil)
			zoneID, recordID, idErr = c.resolveRecordByID(recordValue, client, ctx.Integration.GetMetadata())
			if idErr != nil {
				return fmt.Errorf("resolve record by name: %w", nameErr)
			}
		}
	}

	deleted, err := client.DeleteDNSRecord(zoneID, recordID)
	if err != nil {
		if apiErr := (*CloudflareAPIError)(nil); errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return fmt.Errorf("zone or DNS record not found: %w", err)
		}
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DNSRecordPayloadType, []any{
		map[string]any{
			"zoneId":   zoneID,
			"recordId": deleted.ID,
			"record":   deleted,
		},
	})
}

// parseZoneAndRecordID returns zoneID and recordID. If recordIDValue contains "/" (from
// integration resource selection), it is split into zone and record ID; otherwise zone
// is resolved from zoneValue and recordIDValue is used as the record ID (backward compat).
func (c *DeleteDNSRecord) parseZoneAndRecordID(recordIDValue, zoneValue string, integrationMetadata any) (zoneID, recordID string) {
	if zonePart, recordPart, ok := strings.Cut(recordIDValue, "/"); ok && zonePart != "" && recordPart != "" {
		return zonePart, recordPart
	}
	return c.resolveZoneID(zoneValue, integrationMetadata), recordIDValue
}

// resolveRecordName finds a DNS record by name across zones and returns zoneID and recordID.
// Name matching is case-insensitive and ignores a trailing dot (Cloudflare may return FQDN with or without it).
func (c *DeleteDNSRecord) resolveRecordName(recordName string, client *Client, integrationMetadata any) (zoneID, recordID string, err error) {
	metadata := Metadata{}
	if decodeErr := mapstructure.Decode(integrationMetadata, &metadata); decodeErr != nil {
		return "", "", fmt.Errorf("failed to decode integration metadata: %w", decodeErr)
	}

	want := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(recordName)), ".")

	for _, zone := range metadata.Zones {
		records, listErr := client.ListDNSRecords(zone.ID)
		if listErr != nil {
			continue
		}
		for _, rec := range records {
			got := strings.TrimSuffix(strings.ToLower(rec.Name), ".")
			if got == want {
				return zone.ID, rec.ID, nil
			}
		}
	}

	return "", "", fmt.Errorf("no DNS record found with name %q in any zone", recordName)
}

// resolveRecordByID finds a DNS record by ID across zones and returns zoneID and recordID.
// This supports expressions like {{ createDnsRecord.data.id }} when the record has no "/".
func (c *DeleteDNSRecord) resolveRecordByID(recordID string, client *Client, integrationMetadata any) (zoneID, foundRecordID string, err error) {
	metadata := Metadata{}
	if decodeErr := mapstructure.Decode(integrationMetadata, &metadata); decodeErr != nil {
		return "", "", fmt.Errorf("failed to decode integration metadata: %w", decodeErr)
	}

	want := strings.TrimSpace(recordID)
	if want == "" {
		return "", "", fmt.Errorf("record ID is empty")
	}

	for _, zone := range metadata.Zones {
		records, listErr := client.ListDNSRecords(zone.ID)
		if listErr != nil {
			continue
		}
		for _, rec := range records {
			if rec.ID == want {
				return zone.ID, rec.ID, nil
			}
		}
	}

	return "", "", fmt.Errorf("no DNS record found with ID %q in any zone", recordID)
}

func (c *DeleteDNSRecord) resolveZoneID(zone string, integrationMetadata any) string {
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

func (c *DeleteDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
