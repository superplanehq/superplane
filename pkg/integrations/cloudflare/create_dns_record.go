package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CreateDNSRecordFailedOutputChannel = "failed"
	DNSRecordPayloadType               = "cloudflare.dnsRecord"
)

var (
	allowedDNSRecordTypes  = []string{"A", "AAAA", "CAA", "CNAME", "MX", "NS", "SRV", "TXT"}
	proxyableDNSRecordTypes = []string{"A", "AAAA", "CNAME"}
	priorityDNSRecordTypes  = []string{"MX", "SRV"}
)

type CreateDNSRecord struct{}

type CreateDNSRecordSpec struct {
	Zone     string `json:"zone"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      *int   `json:"ttl"`
	Proxied  *bool  `json:"proxied"`
	Priority *int   `json:"priority"`
}

func (c *CreateDNSRecord) Name() string {
	return "cloudflare.createDnsRecord"
}

func (c *CreateDNSRecord) Label() string {
	return "Create DNS Record"
}

func (c *CreateDNSRecord) Description() string {
	return "Create a DNS record in a Cloudflare zone"
}

func (c *CreateDNSRecord) Documentation() string {
	return `The Create DNS Record component creates a DNS record in a Cloudflare zone.

## Use Cases

- **Provisioning**: Add records when new environments are created
- **Verification**: Create TXT or CNAME records for domain ownership checks
- **Releases**: Add or update canary or migration records

## Configuration

- **Zone**: Select the Cloudflare zone or enter a domain name
- **Type**: DNS record type (A, AAAA, CNAME, MX, TXT, NS, etc.)
- **Name**: Record name (e.g., ` + "`www`" + `, ` + "`api`" + `, or ` + "`@`" + ` for apex)
- **Content**: Record value (IP, hostname, or text)
- **TTL**: Time-to-live in seconds (use ` + "`1`" + ` for auto)
- **Proxied**: Proxy through Cloudflare (A, AAAA, CNAME only)
- **Priority**: Priority value (MX or SRV only)

## Output Channels

- **Success**: Emits the created DNS record
- **Failed**: Emits when the zone is not found, the record is invalid, or a duplicate exists`
}

func (c *CreateDNSRecord) Icon() string {
	return "cloud"
}

func (c *CreateDNSRecord) Color() string {
	return "orange"
}

func (c *CreateDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  core.DefaultOutputChannel.Name,
			Label: "Success",
		},
		{
			Name:  CreateDNSRecordFailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (c *CreateDNSRecord) Configuration() []configuration.Field {
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
			Name:     "type",
			Label:    "Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
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
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Record name (use @ for apex)",
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Record content (IP, hostname, or text value)",
		},
		{
			Name:        "ttl",
			Label:       "TTL",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "TTL in seconds (use 1 for auto)",
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
			Default:     false,
			Description: "Proxy through Cloudflare (A, AAAA, CNAME only)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: proxyableDNSRecordTypes},
			},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Priority for MX or SRV records",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: priorityDNSRecordTypes},
			},
		},
	}
}

func (c *CreateDNSRecord) Setup(ctx core.SetupContext) error {
	spec := CreateDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	recordType := normalizeDNSRecordType(spec.Type)
	if spec.Zone == "" {
		return errors.New("zone is required")
	}
	if recordType == "" {
		return errors.New("type is required")
	}
	if !slices.Contains(allowedDNSRecordTypes, recordType) {
		return fmt.Errorf("type must be one of %s", strings.Join(allowedDNSRecordTypes, ", "))
	}
	if spec.Name == "" {
		return errors.New("name is required")
	}
	if spec.Content == "" {
		return errors.New("content is required")
	}
	if spec.TTL != nil && *spec.TTL < 1 {
		return errors.New("ttl must be greater than 0")
	}
	if spec.Proxied != nil && *spec.Proxied && !slices.Contains(proxyableDNSRecordTypes, recordType) {
		return errors.New("proxied is only supported for A, AAAA, and CNAME records")
	}
	if spec.Priority != nil && !slices.Contains(priorityDNSRecordTypes, recordType) {
		return errors.New("priority is only supported for MX or SRV records")
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

	zoneID := resolveZoneID(spec.Zone, ctx.Integration)
	recordType := normalizeDNSRecordType(spec.Type)

	req := CreateDNSRecordRequest{
		Type:    recordType,
		Name:    spec.Name,
		Content: spec.Content,
		TTL:     spec.TTL,
	}

	if slices.Contains(proxyableDNSRecordTypes, recordType) {
		proxied := false
		if spec.Proxied != nil {
			proxied = *spec.Proxied
		}
		req.Proxied = &proxied
	}

	if slices.Contains(priorityDNSRecordTypes, recordType) {
		req.Priority = spec.Priority
	}

	record, err := client.CreateDNSRecord(zoneID, req)
	if err != nil {
		var apiErr *CloudflareAPIError
		if errors.As(err, &apiErr) && shouldEmitDNSRecordFailure(apiErr) {
			return ctx.ExecutionState.Emit(
				CreateDNSRecordFailedOutputChannel,
				DNSRecordPayloadType,
				[]any{dnsRecordFailurePayload(apiErr)},
			)
		}
		return fmt.Errorf("failed to create DNS record: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DNSRecordPayloadType,
		[]any{dnsRecordToMap(record)},
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
	return http.StatusOK, nil
}

func normalizeDNSRecordType(recordType string) string {
	return strings.ToUpper(strings.TrimSpace(recordType))
}

func resolveZoneID(value string, integration core.IntegrationContext) string {
	if value == "" {
		return value
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return value
	}

	for _, zone := range metadata.Zones {
		if zone.ID == value || zone.Name == value {
			return zone.ID
		}
	}

	return value
}

func shouldEmitDNSRecordFailure(err *CloudflareAPIError) bool {
	switch err.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}

func dnsRecordFailurePayload(err *CloudflareAPIError) map[string]any {
	message := err.Error()
	if len(err.Errors) > 0 {
		message = err.Errors[0].Message
	}

	payload := map[string]any{
		"error":      message,
		"statusCode": err.StatusCode,
	}

	if len(err.Errors) > 0 {
		errorItems := make([]map[string]any, 0, len(err.Errors))
		for _, e := range err.Errors {
			errorItems = append(errorItems, map[string]any{
				"code":    e.Code,
				"message": e.Message,
			})
		}
		payload["errors"] = errorItems
	}

	return payload
}

func dnsRecordToMap(record *DNSRecord) map[string]any {
	payload := map[string]any{
		"id":      record.ID,
		"type":    record.Type,
		"name":    record.Name,
		"content": record.Content,
		"proxied": record.Proxied,
		"ttl":     record.TTL,
	}

	if record.Priority != nil {
		payload["priority"] = *record.Priority
	}

	return payload
}
