package dash0

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_prometheus.json
var exampleOutputQueryPrometheusBytes []byte

var exampleOutputQueryPrometheusOnce sync.Once
var exampleOutputQueryPrometheus map[string]any

//go:embed example_output_list_issues.json
var exampleOutputListIssuesBytes []byte

var exampleOutputListIssuesOnce sync.Once
var exampleOutputListIssues map[string]any

//go:embed example_data_on_alert_event.json
var exampleDataOnAlertEventBytes []byte

var exampleDataOnAlertEventOnce sync.Once
var exampleDataOnAlertEvent map[string]any

//go:embed example_output_send_log_event.json
var exampleOutputSendLogEventBytes []byte

var exampleOutputSendLogEventOnce sync.Once
var exampleOutputSendLogEvent map[string]any

//go:embed example_output_get_check_details.json
var exampleOutputGetCheckDetailsBytes []byte

var exampleOutputGetCheckDetailsOnce sync.Once
var exampleOutputGetCheckDetails map[string]any

//go:embed example_output_create_synthetic_check.json
var exampleOutputCreateSyntheticCheckBytes []byte

var exampleOutputCreateSyntheticCheckOnce sync.Once
var exampleOutputCreateSyntheticCheck map[string]any

//go:embed example_output_update_synthetic_check.json
var exampleOutputUpdateSyntheticCheckBytes []byte

var exampleOutputUpdateSyntheticCheckOnce sync.Once
var exampleOutputUpdateSyntheticCheck map[string]any

//go:embed example_output_create_check_rule.json
var exampleOutputCreateCheckRuleBytes []byte

var exampleOutputCreateCheckRuleOnce sync.Once
var exampleOutputCreateCheckRule map[string]any

//go:embed example_output_update_check_rule.json
var exampleOutputUpdateCheckRuleBytes []byte

var exampleOutputUpdateCheckRuleOnce sync.Once
var exampleOutputUpdateCheckRule map[string]any

// ExampleOutput returns sample output data for Query Prometheus.
func (c *QueryPrometheus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryPrometheusOnce, exampleOutputQueryPrometheusBytes, &exampleOutputQueryPrometheus)
}

// ExampleOutput returns sample output data for List Issues.
func (c *ListIssues) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListIssuesOnce, exampleOutputListIssuesBytes, &exampleOutputListIssues)
}

// ExampleData returns sample webhook payload data for On Alert Event trigger.
func (t *OnAlertEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertEventOnce, exampleDataOnAlertEventBytes, &exampleDataOnAlertEvent)
}

// ExampleOutput returns sample output data for Send Log Event.
func (c *SendLogEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendLogEventOnce, exampleOutputSendLogEventBytes, &exampleOutputSendLogEvent)
}

// ExampleOutput returns sample output data for Get Check Details.
func (c *GetCheckDetails) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCheckDetailsOnce, exampleOutputGetCheckDetailsBytes, &exampleOutputGetCheckDetails)
}

// ExampleOutput returns sample output data for Create Synthetic Check.
func (c *CreateSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSyntheticCheckOnce, exampleOutputCreateSyntheticCheckBytes, &exampleOutputCreateSyntheticCheck)
}

// ExampleOutput returns sample output data for Update Synthetic Check.
func (c *UpdateSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateSyntheticCheckOnce, exampleOutputUpdateSyntheticCheckBytes, &exampleOutputUpdateSyntheticCheck)
}

// ExampleOutput returns sample output data for Create Check Rule.
func (c *CreateCheckRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateCheckRuleOnce, exampleOutputCreateCheckRuleBytes, &exampleOutputCreateCheckRule)
}

// ExampleOutput returns sample output data for Update Check Rule.
func (c *UpdateCheckRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateCheckRuleOnce, exampleOutputUpdateCheckRuleBytes, &exampleOutputUpdateCheckRule)
}
