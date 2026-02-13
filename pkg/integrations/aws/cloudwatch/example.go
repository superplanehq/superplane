package cloudwatch

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alarm.json
var exampleDataOnAlarmBytes []byte

//go:embed example_output_query_metrics_insights.json
var exampleOutputQueryMetricsInsightsBytes []byte

var exampleDataOnAlarmOnce sync.Once
var exampleDataOnAlarm map[string]any

var exampleOutputQueryMetricsInsightsOnce sync.Once
var exampleOutputQueryMetricsInsights map[string]any

func (t *OnAlarm) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlarmOnce, exampleDataOnAlarmBytes, &exampleDataOnAlarm)
}

func (c *QueryMetricsInsights) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputQueryMetricsInsightsOnce,
		exampleOutputQueryMetricsInsightsBytes,
		&exampleOutputQueryMetricsInsights,
	)
}
