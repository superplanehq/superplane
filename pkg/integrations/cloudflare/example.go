package cloudflare

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_update_redirect_rule.json
var exampleOutputUpdateRedirectRuleBytes []byte

var exampleOutputUpdateRedirectRuleOnce sync.Once
var exampleOutputUpdateRedirectRule map[string]any

func (c *UpdateRedirectRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateRedirectRuleOnce, exampleOutputUpdateRedirectRuleBytes, &exampleOutputUpdateRedirectRule)
}

//go:embed example_output_delete_dns_record.json
var exampleOutputDeleteDNSRecordBytes []byte

var exampleOutputDeleteDNSRecordOnce sync.Once
var exampleOutputDeleteDNSRecord map[string]any

func (c *DeleteDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteDNSRecordOnce, exampleOutputDeleteDNSRecordBytes, &exampleOutputDeleteDNSRecord)
}
