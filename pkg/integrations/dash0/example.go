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

//go:embed example_output_create_http_synthetic_check.json
var exampleOutputCreateHTTPSyntheticCheckBytes []byte

var exampleOutputCreateHTTPSyntheticCheckOnce sync.Once
var exampleOutputCreateHTTPSyntheticCheck map[string]any

//go:embed example_output_update_http_synthetic_check.json
var exampleOutputUpdateHTTPSyntheticCheckBytes []byte

var exampleOutputUpdateHTTPSyntheticCheckOnce sync.Once
var exampleOutputUpdateHTTPSyntheticCheck map[string]any

//go:embed example_output_delete_http_synthetic_check.json
var exampleOutputDeleteHTTPSyntheticCheckBytes []byte

var exampleOutputDeleteHTTPSyntheticCheckOnce sync.Once
var exampleOutputDeleteHTTPSyntheticCheck map[string]any

//go:embed example_data_on_alert_notification.json
var exampleDataOnAlertNotificationBytes []byte

//go:embed example_output_get_http_synthetic_check.json
var exampleOutputGetHTTPSyntheticCheckBytes []byte

var exampleOutputGetHTTPSyntheticCheckOnce sync.Once
var exampleOutputGetHTTPSyntheticCheck map[string]any

var exampleDataOnAlertNotificationOnce sync.Once
var exampleDataOnAlertNotification map[string]any

//go:embed example_data_on_synthetic_check_notification.json
var exampleDataOnSyntheticCheckNotificationBytes []byte

var exampleDataOnSyntheticCheckNotificationOnce sync.Once
var exampleDataOnSyntheticCheckNotification map[string]any

//go:embed example_output_create_check_rule.json
var exampleOutputCreateCheckRuleBytes []byte

var exampleOutputCreateCheckRuleOnce sync.Once
var exampleOutputCreateCheckRule map[string]any

//go:embed example_output_get_check_rule.json
var exampleOutputGetCheckRuleBytes []byte

var exampleOutputGetCheckRuleOnce sync.Once
var exampleOutputGetCheckRule map[string]any

//go:embed example_output_update_check_rule.json
var exampleOutputUpdateCheckRuleBytes []byte

var exampleOutputUpdateCheckRuleOnce sync.Once
var exampleOutputUpdateCheckRule map[string]any

//go:embed example_output_delete_check_rule.json
var exampleOutputDeleteCheckRuleBytes []byte

var exampleOutputDeleteCheckRuleOnce sync.Once
var exampleOutputDeleteCheckRule map[string]any

func (c *QueryPrometheus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryPrometheusOnce, exampleOutputQueryPrometheusBytes, &exampleOutputQueryPrometheus)
}

func (c *ListIssues) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListIssuesOnce, exampleOutputListIssuesBytes, &exampleOutputListIssues)
}

func (c *CreateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateHTTPSyntheticCheckOnce, exampleOutputCreateHTTPSyntheticCheckBytes, &exampleOutputCreateHTTPSyntheticCheck)
}

func (c *UpdateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateHTTPSyntheticCheckOnce, exampleOutputUpdateHTTPSyntheticCheckBytes, &exampleOutputUpdateHTTPSyntheticCheck)
}

func (c *DeleteHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteHTTPSyntheticCheckOnce, exampleOutputDeleteHTTPSyntheticCheckBytes, &exampleOutputDeleteHTTPSyntheticCheck)
}

func (c *GetHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetHTTPSyntheticCheckOnce, exampleOutputGetHTTPSyntheticCheckBytes, &exampleOutputGetHTTPSyntheticCheck)
}

func onAlertNotificationExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertNotificationOnce, exampleDataOnAlertNotificationBytes, &exampleDataOnAlertNotification)
}

func onSyntheticCheckNotificationExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnSyntheticCheckNotificationOnce, exampleDataOnSyntheticCheckNotificationBytes, &exampleDataOnSyntheticCheckNotification)
}

func (c *CreateCheckRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateCheckRuleOnce, exampleOutputCreateCheckRuleBytes, &exampleOutputCreateCheckRule)
}

func (c *GetCheckRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCheckRuleOnce, exampleOutputGetCheckRuleBytes, &exampleOutputGetCheckRule)
}

func (c *UpdateCheckRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateCheckRuleOnce, exampleOutputUpdateCheckRuleBytes, &exampleOutputUpdateCheckRule)
}

func (c *DeleteCheckRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteCheckRuleOnce, exampleOutputDeleteCheckRuleBytes, &exampleOutputDeleteCheckRule)
}
