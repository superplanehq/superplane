package cloudwatch

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alarm.json
var exampleDataOnAlarmBytes []byte

var exampleDataOnAlarmOnce sync.Once
var exampleDataOnAlarm map[string]any

func (t *OnAlarm) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlarmOnce, exampleDataOnAlarmBytes, &exampleDataOnAlarm)
}

//go:embed example_output_create_alarm.json
var exampleOutputCreateAlarmBytes []byte

var exampleOutputCreateAlarmOnce sync.Once
var exampleOutputCreateAlarm map[string]any

func (c *CreateAlarm) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateAlarmOnce, exampleOutputCreateAlarmBytes, &exampleOutputCreateAlarm)
}

//go:embed example_output_update_alarm.json
var exampleOutputUpdateAlarmBytes []byte

var exampleOutputUpdateAlarmOnce sync.Once
var exampleOutputUpdateAlarm map[string]any

func (c *UpdateAlarm) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateAlarmOnce, exampleOutputUpdateAlarmBytes, &exampleOutputUpdateAlarm)
}

//go:embed example_output_query_logs.json
var exampleOutputQueryLogsBytes []byte

var exampleOutputQueryLogsOnce sync.Once
var exampleOutputQueryLogs map[string]any

func (c *QueryLogs) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryLogsOnce, exampleOutputQueryLogsBytes, &exampleOutputQueryLogs)
}

//go:embed example_output_add_log_event.json
var exampleOutputAddLogEventBytes []byte

var exampleOutputAddLogEventOnce sync.Once
var exampleOutputAddLogEvent map[string]any

func (c *AddLogEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddLogEventOnce, exampleOutputAddLogEventBytes, &exampleOutputAddLogEvent)
}

//go:embed example_output_get_metric_data.json
var exampleOutputGetMetricDataBytes []byte

var exampleOutputGetMetricDataOnce sync.Once
var exampleOutputGetMetricData map[string]any

func (c *GetMetricData) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetMetricDataOnce, exampleOutputGetMetricDataBytes, &exampleOutputGetMetricData)
}

//go:embed example_output_send_metric_data.json
var exampleOutputSendMetricDataBytes []byte

var exampleOutputSendMetricDataOnce sync.Once
var exampleOutputSendMetricData map[string]any

func (c *SendMetricData) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendMetricDataOnce, exampleOutputSendMetricDataBytes, &exampleOutputSendMetricData)
}
