package route53

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// ChangeInfo contains information about a Route 53 change request.
type ChangeInfo struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submittedAt"`
}

// RecordChangePollMetadata is stored when a change is PENDING and we schedule a poll.
type RecordChangePollMetadata struct {
	ChangeID    string `json:"changeId" mapstructure:"changeId"`
	RecordName  string `json:"recordName" mapstructure:"recordName"`
	RecordType  string `json:"recordType" mapstructure:"recordType"`
	SubmittedAt string `json:"submittedAt" mapstructure:"submittedAt"`
}

// HostedZoneSummary contains basic information about a hosted zone.
type HostedZoneSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ResourceRecordSet represents a DNS record set to be created, updated, or deleted.
type ResourceRecordSet struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	TTL    int      `json:"ttl"`
	Values []string `json:"values"`
}

/*
 * DNS record types supported by AWS Route 53.
 * See: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ResourceRecordTypes.html
 */
var RecordTypeOptions = []configuration.FieldOption{
	{Label: "A", Value: "A"},
	{Label: "AAAA", Value: "AAAA"},
	{Label: "CAA", Value: "CAA"},
	{Label: "CNAME", Value: "CNAME"},
	{Label: "DS", Value: "DS"},
	{Label: "MX", Value: "MX"},
	{Label: "NAPTR", Value: "NAPTR"},
	{Label: "NS", Value: "NS"},
	{Label: "PTR", Value: "PTR"},
	{Label: "SOA", Value: "SOA"},
	{Label: "SPF", Value: "SPF"},
	{Label: "SRV", Value: "SRV"},
	{Label: "TXT", Value: "TXT"},
}

/*
 * recordConfigurationFields returns the shared configuration fields
 * for all Route 53 DNS record components.
 */
func recordConfigurationFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "hostedZoneId",
			Label:       "Hosted Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The hosted zone to manage DNS records in",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "route53.hostedZone",
				},
			},
		},
		{
			Name:        "recordName",
			Label:       "Record Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The DNS record name (e.g. example.com or sub.example.com)",
		},
		{
			Name:     "recordType",
			Label:    "Record Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "A",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: RecordTypeOptions,
				},
			},
		},
		{
			Name:        "ttl",
			Label:       "TTL (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     "300",
			Description: "Time to live for the DNS record in seconds",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
					Max: func() *int { max := 2147483647; return &max }(),
				},
			},
		},
		{
			Name:        "values",
			Label:       "Record Values",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "The values for the DNS record (e.g. IP addresses, domain names)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Value",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func validateRecordConfiguration(hostedZoneID, recordName, recordType string, values []string) error {
	if hostedZoneID == "" {
		return fmt.Errorf("hosted zone is required")
	}

	if recordName == "" {
		return fmt.Errorf("record name is required")
	}

	if recordType == "" {
		return fmt.Errorf("record type is required")
	}

	if len(values) == 0 {
		return fmt.Errorf("at least one record value is required")
	}

	return nil
}

func normalizeValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			normalized = append(normalized, v)
		}
	}
	return normalized
}
