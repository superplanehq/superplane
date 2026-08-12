package cloudwatch

import "github.com/superplanehq/superplane/pkg/configuration"

const (
	AlarmStateOK               = "OK"
	AlarmStateAlarm            = "ALARM"
	AlarmStateInsufficientData = "INSUFFICIENT_DATA"
)

const (
	CreateAlarmPayloadType = "aws.cloudwatch.alarm"
	UpdateAlarmPayloadType = "aws.cloudwatch.alarm"
)

const (
	StatisticAverage = "Average"

	defaultAlarmPeriod            = 300
	defaultAlarmEvaluationPeriods = 1
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

// AlarmEC2ActionOptions are the EC2 automations CloudWatch can run itself when
// an alarm fires. Each value maps to arn:aws:automate:<region>:ec2:<value>.
var AlarmEC2ActionOptions = []configuration.FieldOption{
	{Label: "Recover", Value: "recover"},
	{Label: "Reboot", Value: "reboot"},
	{Label: "Stop", Value: "stop"},
	{Label: "Terminate", Value: "terminate"},
}

var AlarmStatisticOptions = []configuration.FieldOption{
	{Label: "Average", Value: StatisticAverage},
	{Label: "Sum", Value: "Sum"},
	{Label: "Minimum", Value: "Minimum"},
	{Label: "Maximum", Value: "Maximum"},
	{Label: "Sample Count", Value: "SampleCount"},
}

var AlarmComparisonOperatorOptions = []configuration.FieldOption{
	{Label: "Greater Than Threshold", Value: "GreaterThanThreshold"},
	{Label: "Greater Than Or Equal To Threshold", Value: "GreaterThanOrEqualToThreshold"},
	{Label: "Less Than Threshold", Value: "LessThanThreshold"},
	{Label: "Less Than Or Equal To Threshold", Value: "LessThanOrEqualToThreshold"},
}

var AlarmTreatMissingDataOptions = []configuration.FieldOption{
	{Label: "Missing", Value: "missing"},
	{Label: "Ignore", Value: "ignore"},
	{Label: "Breaching", Value: "breaching"},
	{Label: "Not Breaching", Value: "notBreaching"},
}

// AlarmUnitOptions are the metric units accepted by PutMetricAlarm.
var AlarmUnitOptions = []configuration.FieldOption{
	{Label: "None", Value: "None"},
	{Label: "Percent", Value: "Percent"},
	{Label: "Count", Value: "Count"},
	{Label: "Count/Second", Value: "Count/Second"},
	{Label: "Seconds", Value: "Seconds"},
	{Label: "Microseconds", Value: "Microseconds"},
	{Label: "Milliseconds", Value: "Milliseconds"},
	{Label: "Bytes", Value: "Bytes"},
	{Label: "Kilobytes", Value: "Kilobytes"},
	{Label: "Megabytes", Value: "Megabytes"},
	{Label: "Gigabytes", Value: "Gigabytes"},
	{Label: "Terabytes", Value: "Terabytes"},
	{Label: "Bits", Value: "Bits"},
	{Label: "Kilobits", Value: "Kilobits"},
	{Label: "Megabits", Value: "Megabits"},
	{Label: "Gigabits", Value: "Gigabits"},
	{Label: "Terabits", Value: "Terabits"},
	{Label: "Bytes/Second", Value: "Bytes/Second"},
	{Label: "Kilobytes/Second", Value: "Kilobytes/Second"},
	{Label: "Megabytes/Second", Value: "Megabytes/Second"},
	{Label: "Gigabytes/Second", Value: "Gigabytes/Second"},
	{Label: "Terabytes/Second", Value: "Terabytes/Second"},
	{Label: "Bits/Second", Value: "Bits/Second"},
	{Label: "Kilobits/Second", Value: "Kilobits/Second"},
	{Label: "Megabits/Second", Value: "Megabits/Second"},
	{Label: "Gigabits/Second", Value: "Gigabits/Second"},
	{Label: "Terabits/Second", Value: "Terabits/Second"},
}
