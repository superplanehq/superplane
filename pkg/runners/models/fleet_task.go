package models

import "encoding/json"

// FleetTask is the task payload returned by fleet-manager (GET /v1/tasks/:id and webhook body).
type FleetTask struct {
	TaskID   string          `json:"task_id"`
	Status   string          `json:"status"`
	ExitCode *int            `json:"exit_code,omitempty"`
	Output   string          `json:"output,omitempty"`
	Error    string          `json:"error,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`

	TaskLog *FleetTaskLog `json:"task_log,omitempty"`

	CloudWatchLogGroup  string `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream string `json:"cloudwatch_log_stream,omitempty"`
}

func (t *FleetTask) EffectiveExitCode() int {
	if t == nil || t.ExitCode == nil {
		return 0
	}
	return *t.ExitCode
}

func (t *FleetTask) IsInTerminalState() bool {
	return t.Status == "succeeded" || t.Status == "failed" || t.Status == "canceled"
}

// FleetTaskLog matches the fleet-manager JSON shape for CloudWatch-backed live logs.
type FleetTaskLog struct {
	Type       string `json:"type"`
	CloudWatch *struct {
		LogGroupName  string `json:"log_group_name"`
		LogStreamName string `json:"log_stream_name"`
		Region        string `json:"region,omitempty"`
	} `json:"cloudwatch,omitempty"`
}
