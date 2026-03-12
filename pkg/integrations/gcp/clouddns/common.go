package clouddns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

// ChangeInfo contains information about a Cloud DNS change.
type ChangeInfo struct {
	ID        string `json:"id" mapstructure:"id"`
	Status    string `json:"status" mapstructure:"status"`
	StartTime string `json:"startTime" mapstructure:"startTime"`
}

// ResourceRecordSet represents a DNS record set in Cloud DNS.
type ResourceRecordSet struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl"`
	Rrdatas []string `json:"rrdatas"`
}

// RecordSetPollMetadata is stored when a change is pending and we schedule a poll.
type RecordSetPollMetadata struct {
	ChangeID    string `json:"changeId" mapstructure:"changeId"`
	ManagedZone string `json:"managedZone" mapstructure:"managedZone"`
	RecordName  string `json:"recordName" mapstructure:"recordName"`
	RecordType  string `json:"recordType" mapstructure:"recordType"`
	StartTime   string `json:"startTime" mapstructure:"startTime"`
}

var RecordTypeOptions = []configuration.FieldOption{
	{Label: "A", Value: "A"},
	{Label: "AAAA", Value: "AAAA"},
	{Label: "CAA", Value: "CAA"},
	{Label: "CNAME", Value: "CNAME"},
	{Label: "MX", Value: "MX"},
	{Label: "NAPTR", Value: "NAPTR"},
	{Label: "NS", Value: "NS"},
	{Label: "PTR", Value: "PTR"},
	{Label: "SPF", Value: "SPF"},
	{Label: "SRV", Value: "SRV"},
	{Label: "TXT", Value: "TXT"},
}

func baseRecordConfigurationFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "managedZone",
			Label:       "Managed Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud DNS managed zone to manage records in.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeManagedZone,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "name",
			Label:       "Record Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The DNS record name (e.g. api.example.com). A trailing dot will be added automatically.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "managedZone", Values: []string{"*"}},
			},
		},
		{
			Name:     "type",
			Label:    "Record Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "A",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "managedZone", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: RecordTypeOptions,
				},
			},
		},
	}
}

func ttlConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        "ttl",
		Label:       "TTL (seconds)",
		Type:        configuration.FieldTypeNumber,
		Required:    true,
		Default:     "300",
		Description: "Time to live for the DNS record in seconds.",
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "managedZone", Values: []string{"*"}},
		},
		TypeOptions: &configuration.TypeOptions{
			Number: &configuration.NumberTypeOptions{
				Min: func() *int { min := 1; return &min }(),
				Max: func() *int { max := 2147483647; return &max }(),
			},
		},
	}
}

func rrdatasConfigurationField() configuration.Field {
	return configuration.Field{
		Name:        "rrdatas",
		Label:       "Record Values",
		Type:        configuration.FieldTypeList,
		Required:    true,
		Description: "The values for the DNS record (e.g. IP addresses for A records, domain names for CNAME).",
		VisibilityConditions: []configuration.VisibilityCondition{
			{Field: "managedZone", Values: []string{"*"}},
		},
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Value",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeString,
				},
			},
		},
	}
}

func validateBaseConfig(managedZone, name, recordType string) error {
	if managedZone == "" {
		return fmt.Errorf("managed zone is required")
	}
	if name == "" {
		return fmt.Errorf("record name is required")
	}
	if recordType == "" {
		return fmt.Errorf("record type is required")
	}
	return nil
}

func validateRrdatas(rrdatas []string) error {
	if len(rrdatas) == 0 {
		return fmt.Errorf("at least one record value is required")
	}
	return nil
}

// normalizeRecordName ensures the DNS name ends with a trailing dot.
func normalizeRecordName(name string) string {
	name = strings.TrimSpace(name)
	if name != "" && !strings.HasSuffix(name, ".") {
		name += "."
	}
	return name
}

func normalizeRrdatas(rrdatas []string) []string {
	normalized := make([]string, 0, len(rrdatas))
	for _, v := range rrdatas {
		v = strings.TrimSpace(v)
		if v != "" {
			normalized = append(normalized, v)
		}
	}
	return normalized
}

// applyChange posts a change to Cloud DNS and returns the ChangeInfo.
func applyChange(ctx context.Context, client Client, projectID, managedZone string, additions, deletions []ResourceRecordSet) (*ChangeInfo, error) {
	url := fmt.Sprintf("%s/projects/%s/managedZones/%s/changes", cloudDNSBaseURL, projectID, managedZone)
	body := map[string]any{}
	if len(additions) > 0 {
		body["additions"] = additions
	}
	if len(deletions) > 0 {
		body["deletions"] = deletions
	}

	data, err := client.PostURL(ctx, url, body)
	if err != nil {
		return nil, err
	}

	var change ChangeInfo
	if err := json.Unmarshal(data, &change); err != nil {
		return nil, fmt.Errorf("failed to parse change response: %w", err)
	}
	return &change, nil
}

// getChange fetches the current status of a change.
func getChange(ctx context.Context, client Client, projectID, managedZone, changeID string) (*ChangeInfo, error) {
	url := fmt.Sprintf("%s/projects/%s/managedZones/%s/changes/%s", cloudDNSBaseURL, projectID, managedZone, changeID)
	data, err := client.GetURL(ctx, url)
	if err != nil {
		return nil, err
	}

	var change ChangeInfo
	if err := json.Unmarshal(data, &change); err != nil {
		return nil, fmt.Errorf("failed to parse change response: %w", err)
	}
	return &change, nil
}

// getRecordSet fetches an existing record set by name and type, or nil if not found.
type rrsetListResponse struct {
	Rrsets []ResourceRecordSet `json:"rrsets"`
}

func getRecordSet(ctx context.Context, client Client, projectID, managedZone, name, recordType string) (*ResourceRecordSet, error) {
	query := url.Values{
		"name": {name},
		"type": {recordType},
	}
	fullURL := fmt.Sprintf(
		"%s/projects/%s/managedZones/%s/rrsets?%s",
		cloudDNSBaseURL, projectID, managedZone, query.Encode(),
	)
	data, err := client.GetURL(ctx, fullURL)
	if err != nil {
		return nil, err
	}

	var resp rrsetListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse rrsets response: %w", err)
	}

	if len(resp.Rrsets) == 0 {
		return nil, nil
	}
	return &resp.Rrsets[0], nil
}

// listRecordSetsByName fetches all record sets for a given name regardless of type.
func listRecordSetsByName(ctx context.Context, client Client, projectID, managedZone, name string) ([]ResourceRecordSet, error) {
	query := url.Values{
		"name": {name},
	}
	fullURL := fmt.Sprintf(
		"%s/projects/%s/managedZones/%s/rrsets?%s",
		cloudDNSBaseURL, projectID, managedZone, query.Encode(),
	)
	data, err := client.GetURL(ctx, fullURL)
	if err != nil {
		return nil, err
	}

	var resp rrsetListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse rrsets response: %w", err)
	}

	return resp.Rrsets, nil
}
