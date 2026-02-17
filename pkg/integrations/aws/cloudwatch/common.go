package cloudwatch

import "github.com/superplanehq/superplane/pkg/configuration"

const (
	AlarmStateOK               = "OK"
	AlarmStateAlarm            = "ALARM"
	AlarmStateInsufficientData = "INSUFFICIENT_DATA"
)

var AllAlarmStates = []configuration.FieldOption{
	{
		Label: "OK",
		Value: AlarmStateOK,
	},
	{
		Label: "ALARM",
		Value: AlarmStateAlarm,
	},
	{
		Label: "INSUFFICIENT_DATA",
		Value: AlarmStateInsufficientData,
	},
}
