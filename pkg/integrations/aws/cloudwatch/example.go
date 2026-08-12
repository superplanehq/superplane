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
