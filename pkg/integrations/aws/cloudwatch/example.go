package cloudwatch

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alarm.json
var exampleDataOnAlarmBytes []byte

//go:embed example_output_put_metric_data.json
var exampleOutputPutMetricDataBytes []byte

var exampleDataOnAlarmOnce sync.Once
var exampleDataOnAlarm map[string]any

var exampleOutputPutMetricDataOnce sync.Once
var exampleOutputPutMetricData map[string]any

func (t *OnAlarm) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlarmOnce, exampleDataOnAlarmBytes, &exampleDataOnAlarm)
}

func (c *PutMetricData) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPutMetricDataOnce,
		exampleOutputPutMetricDataBytes,
		&exampleOutputPutMetricData,
	)
}
