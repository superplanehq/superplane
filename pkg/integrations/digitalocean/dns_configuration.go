package digitalocean

import "github.com/superplanehq/superplane/pkg/configuration"

var dnsRecordTypeOptions = []configuration.FieldOption{
	{Label: "A", Value: "A"},
	{Label: "AAAA", Value: "AAAA"},
	{Label: "CAA", Value: "CAA"},
	{Label: "CNAME", Value: "CNAME"},
	{Label: "MX", Value: "MX"},
	{Label: "NS", Value: "NS"},
	{Label: "SRV", Value: "SRV"},
	{Label: "TXT", Value: "TXT"},
}

var validDNSRecordTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"CAA":   true,
	"CNAME": true,
	"MX":    true,
	"NS":    true,
	"SRV":   true,
	"TXT":   true,
}

func isValidDNSRecordType(t string) bool {
	return validDNSRecordTypes[t]
}

func dnsRecordConfiguration() []configuration.Field {
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
