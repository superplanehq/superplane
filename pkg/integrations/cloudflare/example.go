package cloudflare

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_update_redirect_rule.json
var exampleOutputUpdateRedirectRuleBytes []byte

//go:embed example_output_create_dns_record.json
var exampleOutputCreateDNSRecordBytes []byte

//go:embed example_output_create_monitor.json
var exampleOutputCreateMonitorBytes []byte

//go:embed example_output_delete_monitor.json
var exampleOutputDeleteMonitorBytes []byte

//go:embed example_data_on_load_balancing_health_alert.json
var exampleDataOnLoadBalancingHealthAlertBytes []byte

var exampleOutputUpdateRedirectRuleOnce sync.Once
var exampleOutputUpdateRedirectRule map[string]any

var exampleOutputCreateDNSRecordOnce sync.Once
var exampleOutputCreateDNSRecord map[string]any

var exampleOutputCreateMonitorOnce sync.Once
var exampleOutputCreateMonitor map[string]any

var exampleOutputDeleteMonitorOnce sync.Once
var exampleOutputDeleteMonitor map[string]any

var exampleDataOnLoadBalancingHealthAlertOnce sync.Once
var exampleDataOnLoadBalancingHealthAlert map[string]any

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDNSRecordOnce, exampleOutputCreateDNSRecordBytes, &exampleOutputCreateDNSRecord)
}

func (c *UpdateRedirectRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateRedirectRuleOnce, exampleOutputUpdateRedirectRuleBytes, &exampleOutputUpdateRedirectRule)
}

func (c *CreateMonitor) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateMonitorOnce, exampleOutputCreateMonitorBytes, &exampleOutputCreateMonitor)
}

func (c *DeleteMonitor) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteMonitorOnce, exampleOutputDeleteMonitorBytes, &exampleOutputDeleteMonitor)
}

func (t *OnLoadBalancingHealthAlert) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnLoadBalancingHealthAlertOnce, exampleDataOnLoadBalancingHealthAlertBytes, &exampleDataOnLoadBalancingHealthAlert)
}

//go:embed example_output_update_dns_record.json
var exampleOutputUpdateDNSRecordBytes []byte

var exampleOutputUpdateDNSRecordOnce sync.Once
var exampleOutputUpdateDNSRecord map[string]any

func (c *UpdateDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateDNSRecordOnce, exampleOutputUpdateDNSRecordBytes, &exampleOutputUpdateDNSRecord)
}

//go:embed example_output_delete_dns_record.json
var exampleOutputDeleteDNSRecordBytes []byte

var exampleOutputDeleteDNSRecordOnce sync.Once
var exampleOutputDeleteDNSRecord map[string]any

func (c *DeleteDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteDNSRecordOnce, exampleOutputDeleteDNSRecordBytes, &exampleOutputDeleteDNSRecord)
}
