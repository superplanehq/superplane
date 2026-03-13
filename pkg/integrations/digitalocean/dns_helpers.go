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
