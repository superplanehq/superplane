package api

import (
	"strconv"

	"github.com/superplane/runner/shared/models"
)

// DefaultExecutionTimeoutSeconds is used when create omits execution_timeout_seconds
// (runner wall-clock limit and fleet-manager lease lower bound for COALESCE).
const DefaultExecutionTimeoutSeconds = 540 // 9 minutes

// MaxExecutionTimeoutSecondsRequest is the largest execution_timeout_seconds accepted on create.
const MaxExecutionTimeoutSecondsRequest = 86400 // 24 hours

// LeaseBufferSeconds is added to the execution window when computing claim lease_until
// (complete RPC, clock skew).
const LeaseBufferSeconds = 90

// CreateTaskRequest is POST /v1/tasks.
type CreateTaskRequest struct {
	// Command is argv for one process. Omit when using Commands.
	Command []string `json:"command,omitempty"`
	// Commands are shell directives (non-empty trimmed lines). The runner uses Bash on
	// a PTY and sources each directive from a tempfile, stopping at the first failing
	// directive ($? after source). Omit when using Command.
	Commands      []string `json:"commands,omitempty"`
	WebhookURL    string   `json:"webhook_url"`
	ExecutionMode string   `json:"execution_mode"` // "host" | "docker"
	DockerImage   string   `json:"docker_image,omitempty"`
	// ExecutionTimeoutSeconds is optional wall-clock limit for the runner execution phase (seconds).
	// Omit to use DefaultExecutionTimeoutSeconds on the runner; if set, must be 1..MaxExecutionTimeoutSecondsRequest.
	ExecutionTimeoutSeconds *int `json:"execution_timeout_seconds,omitempty"`
}

// CreateTaskResponse returns the task id.
type CreateTaskResponse struct {
	ID string `json:"id"`
}

// ClaimTaskRequest is POST /v1/tasks/claim.
type ClaimTaskRequest struct {
	RunnerID     string `json:"runner_id"`
	LeaseSeconds int    `json:"lease_seconds"`
}

// ClaimTaskResponse returns a task or null task when queue is empty.
type ClaimTaskResponse struct {
	Task *TaskPayload `json:"task"`
}

// TaskPayload is the task spec sent to runners.
type TaskPayload struct {
	ID            string   `json:"id"`
	Command       []string `json:"command,omitempty"`
	Commands      []string `json:"commands,omitempty"` // see CreateTaskRequest (Bash+PTY per directive on Unix)
	ExecutionMode string   `json:"execution_mode"`
	DockerImage   string   `json:"docker_image,omitempty"`
	// ExecutionTimeoutSeconds is nil when unset at create (runner uses default).
	ExecutionTimeoutSeconds *int `json:"execution_timeout_seconds,omitempty"`
}

// CompleteTaskRequest is POST /v1/tasks/{id}/complete.
type CompleteTaskRequest struct {
	RunnerID string `json:"runner_id"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	// Canceled when true means the runner stopped the task due to caller cancel (terminal status canceled).
	Canceled bool `json:"canceled,omitempty"`
}

// TaskPayloadFrom maps models.Task to TaskPayload.
func TaskPayloadFrom(t *models.Task) *TaskPayload {
	p := &TaskPayload{
		ID:            t.ID,
		Command:       t.Command,
		Commands:      t.Commands,
		ExecutionMode: string(t.ExecutionMode),
		DockerImage:   t.DockerImage,
	}
	if t.ExecutionTimeoutSeconds != nil {
		v := *t.ExecutionTimeoutSeconds
		p.ExecutionTimeoutSeconds = &v
	}
	return p
}

// ValidateExecutionTimeoutSeconds returns a non-empty error message if p is set but out of range.
func ValidateExecutionTimeoutSeconds(p *int) string {
	if p == nil {
		return ""
	}
	v := *p
	if v < 1 || v > MaxExecutionTimeoutSecondsRequest {
		return "execution_timeout_seconds must be between 1 and " + strconv.Itoa(MaxExecutionTimeoutSecondsRequest)
	}
	return ""
}

// WebhookPayload is POSTed to the caller webhook URL on terminal status.
type WebhookPayload struct {
	TaskID      string `json:"task_id"`
	FleetTaskID string `json:"fleet_task_id,omitempty"` // when task-broker forwards, fleet-managed id
	Status      string `json:"status"`
	ExitCode    int    `json:"exit_code"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
	// CloudWatch fields mirror TaskStatusResponse when fleet-manager advertises log routing.
	CloudWatchLogGroup  string `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream string `json:"cloudwatch_log_stream,omitempty"`
	// TaskLog is set when an external log sink is configured (e.g. type "cloudwatch"). CloudWatch* fields remain for backward compatibility.
	TaskLog *TaskLogSink `json:"task_log,omitempty"`
}

// TaskStatusResponse is GET fleet-manager /v1/tasks/{id}.
type TaskStatusResponse struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	ExitCode        *int   `json:"exit_code,omitempty"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
	CancelRequested bool   `json:"cancel_requested,omitempty"`
	// CloudWatchLogGroup and CloudWatchLogStream are set when fleet-manager is configured
	// with TASK_CLOUDWATCH_LOG_GROUP so clients can tail logs in AWS (runner must use the same group/prefix).
	CloudWatchLogGroup      string       `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream     string       `json:"cloudwatch_log_stream,omitempty"`
	TaskLog                 *TaskLogSink `json:"task_log,omitempty"`
	ExecutionTimeoutSeconds *int         `json:"execution_timeout_seconds,omitempty"`
}

// CancelTaskResponse is POST fleet-manager /v1/tasks/{id}/cancel.
type CancelTaskResponse struct {
	ID     string `json:"id"`
	State  string `json:"state"`  // already_terminal | canceled | cancel_requested
	Status string `json:"status"` // task status after the operation
}

// BrokerGetTaskResponse is GET task-broker /v1/tasks/{broker_task_id}.
type BrokerGetTaskResponse struct {
	TaskID                  string       `json:"task_id"`
	FleetTaskID             string       `json:"fleet_task_id,omitempty"`
	Status                  string       `json:"status"`
	ExitCode                *int         `json:"exit_code,omitempty"`
	Output                  string       `json:"output,omitempty"`
	Error                   string       `json:"error,omitempty"`
	CancelRequested         bool         `json:"cancel_requested,omitempty"`
	CloudWatchLogGroup      string       `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream     string       `json:"cloudwatch_log_stream,omitempty"`
	TaskLog                 *TaskLogSink `json:"task_log,omitempty"`
	ExecutionTimeoutSeconds *int         `json:"execution_timeout_seconds,omitempty"`
}
