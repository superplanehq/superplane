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
