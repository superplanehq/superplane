package newrelic

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_report_metric.json
var exampleOutputReportMetricBytes []byte

//go:embed example_output_run_nrql_query.json
var exampleOutputRunNRQLQueryBytes []byte

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte
var exampleOutputReportMetric = utils.NewEmbeddedJSON(exampleOutputReportMetricBytes)
var exampleOutputRunNRQLQuery = utils.NewEmbeddedJSON(exampleOutputRunNRQLQueryBytes)
var exampleDataOnIssue = utils.NewEmbeddedJSON(exampleDataOnIssueBytes)

func (c *ReportMetric) ExampleOutput() map[string]any {
	return exampleOutputReportMetric.Value()
}

func (c *RunNRQLQuery) ExampleOutput() map[string]any {
	return exampleOutputRunNRQLQuery.Value()
}

func (t *OnIssue) ExampleData() map[string]any {
	return exampleDataOnIssue.Value()
}
