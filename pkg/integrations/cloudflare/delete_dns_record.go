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

const (
	DNSRecordPayloadType          = "cloudflare.dnsRecord"
	DNSRecordDeleteSuccessChannel = "success"
	DNSRecordDeleteFailedChannel  = "failed"
)

type DeleteDNSRecord struct{}

type DeleteDNSRecordSpec struct {
	Zone     string `json:"zone"`
	RecordID string `json:"recordId"`
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

- **Zone**: Cloudflare zone (zone ID or domain name)
- **Record ID**: DNS record ID to delete

## Output Channels

- **Success**: Emitted when the DNS record is deleted
- **Failed**: Emitted when the zone/record is not found or when deletion fails`
}

func (c *DeleteDNSRecord) Icon() string {
	return "cloud"
}

func (c *DeleteDNSRecord) Color() string {
	return "orange"
}

func (c *DeleteDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  DNSRecordDeleteSuccessChannel,
			Label: "Success",
		},
		{
			Name:  DNSRecordDeleteFailedChannel,
			Label: "Failed",
		},
	}
}

func (c *DeleteDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare zone (zone ID or domain name)",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "zone",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "recordId",
			Label:       "Record ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The DNS record ID to delete",
		},
	}
}

func (c *DeleteDNSRecord) Setup(ctx core.SetupContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Zone) == "" {
		return errors.New("zone is required")
	}

	if strings.TrimSpace(spec.RecordID) == "" {
		return errors.New("recordId is required")
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

	zoneID, resolved := resolveZoneIDFromMetadata(ctx.Integration, spec.Zone)
	if !resolved && strings.Contains(spec.Zone, ".") {
		return ctx.ExecutionState.Emit(
			DNSRecordDeleteFailedChannel,
			DNSRecordPayloadType,
			[]any{
				map[string]any{
					"success":  false,
					"zone":     spec.Zone,
					"recordId": spec.RecordID,
					"error":    "zone not found in integration metadata",
				},
			},
		)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	result, err := client.DeleteDNSRecord(zoneID, spec.RecordID)
	if err != nil {
		statusCode := 0
		if apiErr := (*APIError)(nil); errors.As(err, &apiErr) {
			statusCode = apiErr.StatusCode
		}

		message := "failed to delete DNS record"
		if statusCode == http.StatusNotFound {
			message = "zone or DNS record not found"
		}

		return ctx.ExecutionState.Emit(
			DNSRecordDeleteFailedChannel,
			DNSRecordPayloadType,
			[]any{
				map[string]any{
					"success":    false,
					"zone":       spec.Zone,
					"zoneId":     zoneID,
					"recordId":   spec.RecordID,
					"error":      message,
					"statusCode": statusCode,
					"details":    err.Error(),
				},
			},
		)
	}

	return ctx.ExecutionState.Emit(
		DNSRecordDeleteSuccessChannel,
		DNSRecordPayloadType,
		[]any{
			map[string]any{
				"success":  true,
				"message":  "DNS record deleted",
				"zone":     spec.Zone,
				"zoneId":   zoneID,
				"recordId": spec.RecordID,
				"record":   result,
			},
		},
	)
}

func resolveZoneIDFromMetadata(integration core.IntegrationContext, zone string) (string, bool) {
	// Best effort: if metadata exists and has a matching zone by name or id, map to ID.
	metadataRaw := integration.GetMetadata()
	if metadataRaw == nil {
		return zone, false
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(metadataRaw, &metadata); err != nil {
		return zone, false
	}

	for _, z := range metadata.Zones {
		if z.ID == zone || strings.EqualFold(z.Name, zone) {
			return z.ID, true
		}
	}

	return zone, false
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

