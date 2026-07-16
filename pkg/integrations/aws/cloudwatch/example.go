package cloudwatch

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alarm.json
var exampleDataOnAlarmBytes []byte
var exampleDataOnAlarm = utils.NewEmbeddedJSON(exampleDataOnAlarmBytes)

func (t *OnAlarm) ExampleData() map[string]any {
	return exampleDataOnAlarm.Value()
}
