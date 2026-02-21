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

//go:embed example_data_on_notification.json
var exampleDataOnNotificationBytes []byte

var exampleDataOnNotificationOnce sync.Once
var exampleDataOnNotification map[string]any

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

func onNotificationExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnNotificationOnce, exampleDataOnNotificationBytes, &exampleDataOnNotification)
}
