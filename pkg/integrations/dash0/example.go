package dash0

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_prometheus.json
var exampleOutputQueryPrometheusBytes []byte
var exampleOutputQueryPrometheus = utils.NewEmbeddedJSON(exampleOutputQueryPrometheusBytes)

//go:embed example_output_list_issues.json
var exampleOutputListIssuesBytes []byte
var exampleOutputListIssues = utils.NewEmbeddedJSON(exampleOutputListIssuesBytes)

//go:embed example_output_create_http_synthetic_check.json
var exampleOutputCreateHTTPSyntheticCheckBytes []byte
var exampleOutputCreateHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputCreateHTTPSyntheticCheckBytes)

//go:embed example_output_update_http_synthetic_check.json
var exampleOutputUpdateHTTPSyntheticCheckBytes []byte
var exampleOutputUpdateHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputUpdateHTTPSyntheticCheckBytes)

//go:embed example_output_delete_http_synthetic_check.json
var exampleOutputDeleteHTTPSyntheticCheckBytes []byte
var exampleOutputDeleteHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputDeleteHTTPSyntheticCheckBytes)

//go:embed example_output_send_log_event.json
var exampleOutputSendLogEventBytes []byte
var exampleOutputSendLogEvent = utils.NewEmbeddedJSON(exampleOutputSendLogEventBytes)

//go:embed example_data_on_alert_notification.json
var exampleDataOnAlertNotificationBytes []byte

//go:embed example_output_get_http_synthetic_check.json
var exampleOutputGetHTTPSyntheticCheckBytes []byte
var exampleOutputGetHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputGetHTTPSyntheticCheckBytes)
var exampleDataOnAlertNotification = utils.NewEmbeddedJSON(exampleDataOnAlertNotificationBytes)

//go:embed example_data_on_synthetic_check_notification.json
var exampleDataOnSyntheticCheckNotificationBytes []byte
var exampleDataOnSyntheticCheckNotification = utils.NewEmbeddedJSON(exampleDataOnSyntheticCheckNotificationBytes)

//go:embed example_output_create_check_rule.json
var exampleOutputCreateCheckRuleBytes []byte
var exampleOutputCreateCheckRule = utils.NewEmbeddedJSON(exampleOutputCreateCheckRuleBytes)

//go:embed example_output_get_check_rule.json
var exampleOutputGetCheckRuleBytes []byte
var exampleOutputGetCheckRule = utils.NewEmbeddedJSON(exampleOutputGetCheckRuleBytes)

//go:embed example_output_update_check_rule.json
var exampleOutputUpdateCheckRuleBytes []byte
var exampleOutputUpdateCheckRule = utils.NewEmbeddedJSON(exampleOutputUpdateCheckRuleBytes)

//go:embed example_output_delete_check_rule.json
var exampleOutputDeleteCheckRuleBytes []byte
var exampleOutputDeleteCheckRule = utils.NewEmbeddedJSON(exampleOutputDeleteCheckRuleBytes)

func (c *QueryPrometheus) ExampleOutput() map[string]any {
	return exampleOutputQueryPrometheus.Value()
}

func (c *ListIssues) ExampleOutput() map[string]any {
	return exampleOutputListIssues.Value()
}

func (c *CreateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputCreateHTTPSyntheticCheck.Value()
}

func (c *UpdateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputUpdateHTTPSyntheticCheck.Value()
}

func (c *DeleteHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputDeleteHTTPSyntheticCheck.Value()
}

func (c *SendLogEvent) ExampleOutput() map[string]any {
	return exampleOutputSendLogEvent.Value()
}

func (c *GetHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputGetHTTPSyntheticCheck.Value()
}

func onAlertNotificationExampleData() map[string]any {
	return exampleDataOnAlertNotification.Value()
}

func onSyntheticCheckNotificationExampleData() map[string]any {
	return exampleDataOnSyntheticCheckNotification.Value()
}

func (c *CreateCheckRule) ExampleOutput() map[string]any {
	return exampleOutputCreateCheckRule.Value()
}

func (c *GetCheckRule) ExampleOutput() map[string]any {
	return exampleOutputGetCheckRule.Value()
}

func (c *UpdateCheckRule) ExampleOutput() map[string]any {
	return exampleOutputUpdateCheckRule.Value()
}

func (c *DeleteCheckRule) ExampleOutput() map[string]any {
	return exampleOutputDeleteCheckRule.Value()
}
