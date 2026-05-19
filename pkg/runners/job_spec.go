package runners

// JobSpec is the work payload SuperPlane queues for fleet-manager to pull.
type JobSpec struct {
	Command                 []string                   `json:"command,omitempty"`
	Commands                []string                   `json:"commands,omitempty"`
	Environment             []FleetEnvironmentVariable `json:"environment,omitempty"`
	ExecutionMode           string                     `json:"execution_mode,omitempty"`
	DockerImage             string                     `json:"docker_image,omitempty"`
	ExecutionTimeoutSeconds *int                       `json:"execution_timeout_seconds,omitempty"`
}

// TaskLogSink describes where to read task logs (e.g. CloudWatch).
type TaskLogSink struct {
	Type       string                 `json:"type"`
	CloudWatch *TaskLogSinkCloudWatch `json:"cloudwatch,omitempty"`
}

// TaskLogSinkCloudWatch identifies a CloudWatch Logs stream.
type TaskLogSinkCloudWatch struct {
	LogGroupName  string `json:"log_group_name"`
	LogStreamName string `json:"log_stream_name"`
	Region        string `json:"region,omitempty"`
}

const (
	TaskStatusQueued     = "queued"
	TaskStatusDispatched = "dispatched"
	TaskStatusRunning    = "running"
	TaskStatusSucceeded  = "succeeded"
	TaskStatusFailed     = "failed"
	TaskStatusCanceled   = "canceled"

	FleetModeBridge = "bridge"
	FleetModePush   = "push"
)
