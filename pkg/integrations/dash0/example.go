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
