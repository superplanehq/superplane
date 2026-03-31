package grafana

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_data_source.json
var exampleOutputQueryDataSourceBytes []byte

//go:embed example_output_create_alert_rule.json
var exampleOutputCreateAlertRuleBytes []byte

//go:embed example_output_get_alert_rule.json
var exampleOutputGetAlertRuleBytes []byte

//go:embed example_output_update_alert_rule.json
var exampleOutputUpdateAlertRuleBytes []byte

//go:embed example_data_on_alert_firing.json
var exampleDataOnAlertFiringBytes []byte

var exampleOutputQueryDataSourceOnce sync.Once
var exampleOutputQueryDataSource map[string]any

var exampleOutputCreateAlertRuleOnce sync.Once
var exampleOutputCreateAlertRule map[string]any

var exampleOutputGetAlertRuleOnce sync.Once
var exampleOutputGetAlertRule map[string]any

var exampleOutputUpdateAlertRuleOnce sync.Once
var exampleOutputUpdateAlertRule map[string]any

var exampleDataOnAlertFiringOnce sync.Once
var exampleDataOnAlertFiring map[string]any

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryDataSourceOnce, exampleOutputQueryDataSourceBytes, &exampleOutputQueryDataSource)
}

func (c *CreateAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateAlertRuleOnce,
		exampleOutputCreateAlertRuleBytes,
		&exampleOutputCreateAlertRule,
	)
}

func (c *GetAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetAlertRuleOnce,
		exampleOutputGetAlertRuleBytes,
		&exampleOutputGetAlertRule,
	)
}

func (c *UpdateAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateAlertRuleOnce,
		exampleOutputUpdateAlertRuleBytes,
		&exampleOutputUpdateAlertRule,
	)
}

func (t *OnAlertFiring) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertFiringOnce, exampleDataOnAlertFiringBytes, &exampleDataOnAlertFiring)
}
