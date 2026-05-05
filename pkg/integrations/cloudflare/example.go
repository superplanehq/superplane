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

var exampleOutputUpdateRedirectRuleOnce sync.Once
var exampleOutputUpdateRedirectRule map[string]any

var exampleOutputCreateDNSRecordOnce sync.Once
var exampleOutputCreateDNSRecord map[string]any

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDNSRecordOnce, exampleOutputCreateDNSRecordBytes, &exampleOutputCreateDNSRecord)
}

func (c *UpdateRedirectRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateRedirectRuleOnce, exampleOutputUpdateRedirectRuleBytes, &exampleOutputUpdateRedirectRule)
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

//go:embed example_output_create_kv_namespace.json
var exampleOutputCreateKVNamespaceBytes []byte

var exampleOutputCreateKVNamespaceOnce sync.Once
var exampleOutputCreateKVNamespace map[string]any

func (c *CreateKVNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateKVNamespaceOnce, exampleOutputCreateKVNamespaceBytes, &exampleOutputCreateKVNamespace)
}

//go:embed example_output_put_kv_value.json
var exampleOutputPutKVValueBytes []byte

var exampleOutputPutKVValueOnce sync.Once
var exampleOutputPutKVValue map[string]any

func (c *PutKVValue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPutKVValueOnce, exampleOutputPutKVValueBytes, &exampleOutputPutKVValue)
}

//go:embed example_output_get_kv_value.json
var exampleOutputGetKVValueBytes []byte

var exampleOutputGetKVValueOnce sync.Once
var exampleOutputGetKVValue map[string]any

func (c *GetKVValue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetKVValueOnce, exampleOutputGetKVValueBytes, &exampleOutputGetKVValue)
}

//go:embed example_output_delete_kv_value.json
var exampleOutputDeleteKVValueBytes []byte

var exampleOutputDeleteKVValueOnce sync.Once
var exampleOutputDeleteKVValue map[string]any

func (c *DeleteKVValue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteKVValueOnce, exampleOutputDeleteKVValueBytes, &exampleOutputDeleteKVValue)
}

//go:embed example_output_delete_kv_namespace.json
var exampleOutputDeleteKVNamespaceBytes []byte

var exampleOutputDeleteKVNamespaceOnce sync.Once
var exampleOutputDeleteKVNamespace map[string]any

func (c *DeleteKVNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteKVNamespaceOnce, exampleOutputDeleteKVNamespaceBytes, &exampleOutputDeleteKVNamespace)
}

//go:embed example_output_create_pool.json
var exampleOutputCreatePoolBytes []byte

var exampleOutputCreatePoolOnce sync.Once
var exampleOutputCreatePool map[string]any

func (c *CreatePool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreatePoolOnce, exampleOutputCreatePoolBytes, &exampleOutputCreatePool)
}

//go:embed example_output_update_pool.json
var exampleOutputUpdatePoolBytes []byte

var exampleOutputUpdatePoolOnce sync.Once
var exampleOutputUpdatePool map[string]any

func (c *UpdatePool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdatePoolOnce, exampleOutputUpdatePoolBytes, &exampleOutputUpdatePool)
}

//go:embed example_output_get_pool.json
var exampleOutputGetPoolBytes []byte

var exampleOutputGetPoolOnce sync.Once
var exampleOutputGetPool map[string]any

func (c *GetPool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPoolOnce, exampleOutputGetPoolBytes, &exampleOutputGetPool)
}

//go:embed example_output_delete_pool.json
var exampleOutputDeletePoolBytes []byte

var exampleOutputDeletePoolOnce sync.Once
var exampleOutputDeletePool map[string]any

func (c *DeletePool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeletePoolOnce, exampleOutputDeletePoolBytes, &exampleOutputDeletePool)
}
