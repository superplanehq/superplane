package newrelic

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

func (t *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

//go:embed example_output_report_metric.json
var exampleOutputReportMetricBytes []byte

var exampleOutputReportMetricOnce sync.Once
var exampleOutputReportMetric map[string]any

func (c *ReportMetric) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputReportMetricOnce, exampleOutputReportMetricBytes, &exampleOutputReportMetric)
}

//go:embed example_output_run_nrql_query.json
var exampleOutputRunNRQLQueryBytes []byte

var exampleOutputRunNRQLQueryOnce sync.Once
var exampleOutputRunNRQLQuery map[string]any

func (c *RunNRQLQuery) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunNRQLQueryOnce, exampleOutputRunNRQLQueryBytes, &exampleOutputRunNRQLQuery)
}
