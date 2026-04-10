package grafana

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_data_source.json
var exampleOutputQueryDataSourceBytes []byte

//go:embed example_data_on_alert_firing.json
var exampleDataOnAlertFiringBytes []byte

//go:embed example_output_list_data_sources.json
var exampleOutputListDataSourcesBytes []byte

//go:embed example_output_get_data_source.json
var exampleOutputGetDataSourceBytes []byte

//go:embed example_output_query_logs.json
var exampleOutputQueryLogsBytes []byte

//go:embed example_output_query_traces.json
var exampleOutputQueryTracesBytes []byte

var exampleOutputQueryDataSourceOnce sync.Once
var exampleOutputQueryDataSource map[string]any

var exampleDataOnAlertFiringOnce sync.Once
var exampleDataOnAlertFiring map[string]any

var exampleOutputListDataSourcesOnce sync.Once
var exampleOutputListDataSources map[string]any

var exampleOutputGetDataSourceOnce sync.Once
var exampleOutputGetDataSource map[string]any

var exampleOutputQueryLogsOnce sync.Once
var exampleOutputQueryLogs map[string]any

var exampleOutputQueryTracesOnce sync.Once
var exampleOutputQueryTraces map[string]any

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryDataSourceOnce, exampleOutputQueryDataSourceBytes, &exampleOutputQueryDataSource)
}

func (t *OnAlertFiring) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertFiringOnce, exampleDataOnAlertFiringBytes, &exampleDataOnAlertFiring)
}

func (l *ListDataSources) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListDataSourcesOnce, exampleOutputListDataSourcesBytes, &exampleOutputListDataSources)
}

func (g *GetDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetDataSourceOnce, exampleOutputGetDataSourceBytes, &exampleOutputGetDataSource)
}

func (q *QueryLogs) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryLogsOnce, exampleOutputQueryLogsBytes, &exampleOutputQueryLogs)
}

func (q *QueryTraces) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryTracesOnce, exampleOutputQueryTracesBytes, &exampleOutputQueryTraces)
}
