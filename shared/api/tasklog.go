package api

import "strings"

// TaskLogSink is a discriminated union: clients switch on Type and read the matching block.
// New backends add a new Type value and an optional typed field.
//
// JSON example:
//
//	{"type":"cloudwatch","cloudwatch":{"log_group_name":"…","log_stream_name":"…","region":"us-east-1"}}
const (
	TaskLogTypeCloudWatch = "cloudwatch"
)

// TaskLogSink describes where to read live or historical task logs.
type TaskLogSink struct {
	Type string `json:"type"`

	CloudWatch *TaskLogSinkCloudWatch `json:"cloudwatch,omitempty"`
}

// TaskLogSinkCloudWatch identifies an Amazon CloudWatch Logs stream.
type TaskLogSinkCloudWatch struct {
	LogGroupName  string `json:"log_group_name"`
	LogStreamName string `json:"log_stream_name"`
	Region        string `json:"region,omitempty"`
}

// TaskLogSinkCloudWatchFromParts returns a sink when group and stream are non-empty; otherwise nil.
func TaskLogSinkCloudWatchFromParts(logGroup, logStream, region string) *TaskLogSink {
	g := strings.TrimSpace(logGroup)
	s := strings.TrimSpace(logStream)
	if g == "" || s == "" {
		return nil
	}
	return &TaskLogSink{
		Type: TaskLogTypeCloudWatch,
		CloudWatch: &TaskLogSinkCloudWatch{
			LogGroupName:  g,
			LogStreamName: s,
			Region:        strings.TrimSpace(region),
		},
	}
}
