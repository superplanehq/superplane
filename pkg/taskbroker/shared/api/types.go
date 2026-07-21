package api

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

// DefaultExecutionTimeoutSeconds is used when create omits execution_timeout_seconds
// (runner wall-clock limit and fleet-manager lease lower bound for COALESCE).
const DefaultExecutionTimeoutSeconds = 3600 // 1 hour

// MaxExecutionTimeoutSecondsRequest is the largest execution_timeout_seconds accepted on create.
const MaxExecutionTimeoutSecondsRequest = 86400 // 24 hours

// LeaseBufferSeconds is added to the execution window when computing claim lease_until
// (complete RPC, clock skew).
const LeaseBufferSeconds = 90

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// EnvironmentVariable is one task-scoped environment variable.
type EnvironmentVariable = models.EnvironmentVariable

// CreateTaskRequest is POST /v1/tasks.
type CreateTaskRequest struct {
	// RunMode is command_list, argv, javascript_script, python_script, or bash_script. Omit to infer from body fields.
	RunMode string `json:"run_mode,omitempty"`
	// Script is user code when run_mode is a script mode.
	Script string `json:"script,omitempty"`
	// MessageChain is the SuperPlane $ object for script tasks.
	MessageChain json.RawMessage `json:"message_chain,omitempty"`
	// Command is argv for one process. Omit when using Commands or Script.
	Command []string `json:"command,omitempty"`
	// Commands are shell directives. Each entry may be a plain string or
	// {"name","command"}. The runner uses Bash on a PTY and sources each directive
	// from a tempfile, stopping at the first failing directive ($? after source).
	// Omit when using Command.
	Commands models.CommandList `json:"commands,omitempty"`
	// SetupCommands are optional shell directives run before script execution.
	SetupCommands []string `json:"setup_commands,omitempty"`
	// Environment is sent only to runners and is not returned in status/webhook payloads.
	Environment []EnvironmentVariable `json:"environment,omitempty"`
	// Files are materialized under SUPERPLANE_TASK_DIR before execution (all run modes).
	Files         []TaskFile `json:"files,omitempty"`
	WebhookURL    string     `json:"webhook_url"`
	ExecutionMode string     `json:"execution_mode"` // "host" | "docker"
	DockerImage   string     `json:"docker_image,omitempty"`
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
	FleetID      string `json:"fleet_id"`
	LeaseSeconds int    `json:"lease_seconds"`
}

// ClaimTaskResponse returns a task or null task when queue is empty.
type ClaimTaskResponse struct {
	Task *TaskPayload `json:"task"`
}

// TaskPayload is the task spec sent to runners.
type TaskPayload struct {
	ID            string                `json:"id"`
	RunMode       string                `json:"run_mode,omitempty"`
	Script        string                `json:"script,omitempty"`
	MessageChain  json.RawMessage       `json:"message_chain,omitempty"`
	Command       []string              `json:"command,omitempty"`
	Commands      models.CommandList    `json:"commands,omitempty"` // see CreateTaskRequest (Bash+PTY per directive on Unix)
	SetupCommands []string              `json:"setup_commands,omitempty"`
	Environment   []EnvironmentVariable `json:"environment,omitempty"`
	Files         []TaskFile            `json:"files,omitempty"`
	ExecutionMode string                `json:"execution_mode"`
	DockerImage   string                `json:"docker_image,omitempty"`
	// ExecutionTimeoutSeconds is nil when unset at create (runner uses default).
	ExecutionTimeoutSeconds *int `json:"execution_timeout_seconds,omitempty"`
}

// CompleteTaskRequest is POST /v1/tasks/{id}/complete.
type CompleteTaskRequest struct {
	RunnerID string `json:"runner_id"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
	// FailureKind identifies runner/infrastructure failures separately from user command failures.
	FailureKind string `json:"failure_kind,omitempty"`
	// Result is optional JSON read by the runner from SUPERPLANE_RESULT_FILE after execution.
	Result json.RawMessage `json:"result,omitempty"`
	// Canceled when true means the runner stopped the task due to caller cancel (terminal status canceled).
	Canceled bool `json:"canceled,omitempty"`
}

const (
	FailureKindRunnerInfra = "runner_infra"
)

// TaskPayloadFrom maps models.Task to TaskPayload.
func TaskPayloadFrom(t *models.Task) *TaskPayload {
	p := &TaskPayload{
		ID:            t.ID,
		RunMode:       string(t.RunMode),
		Script:        t.Script,
		Command:       t.Command,
		Commands:      t.Commands,
		SetupCommands: t.SetupCommands,
		Environment:   CloneEnvironment(t.Environment),
		Files:         CloneFiles(t.Files),
		ExecutionMode: string(t.ExecutionMode),
		DockerImage:   t.DockerImage,
	}
	if mc := strings.TrimSpace(t.MessageChainJSON); mc != "" {
		p.MessageChain = json.RawMessage(mc)
	}
	if t.ExecutionTimeoutSeconds != nil {
		v := *t.ExecutionTimeoutSeconds
		p.ExecutionTimeoutSeconds = &v
	}
	return p
}

// CloneEnvironment returns a detached copy of environment variables.
func CloneEnvironment(env []EnvironmentVariable) []EnvironmentVariable {
	if len(env) == 0 {
		return nil
	}
	out := make([]EnvironmentVariable, len(env))
	copy(out, env)
	return out
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

// ValidateEnvironment returns a non-empty error message when task environment is invalid.
func ValidateEnvironment(env []EnvironmentVariable) string {
	seen := make(map[string]struct{}, len(env))
	for _, variable := range env {
		if !environmentNamePattern.MatchString(variable.Name) {
			return "invalid environment variable name"
		}
		if _, ok := seen[variable.Name]; ok {
			return "duplicate environment variable name"
		}
		seen[variable.Name] = struct{}{}
		if strings.ContainsRune(variable.Value, '\x00') {
			return "environment variable values cannot contain NUL bytes"
		}
	}
	return ""
}

// WebhookPayload is POSTed to the caller webhook URL on terminal status.
type WebhookPayload struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
	// CloudWatch fields mirror TaskStatusResponse when task-broker advertises log routing.
	CloudWatchLogGroup  string `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream string `json:"cloudwatch_log_stream,omitempty"`
	// TaskLog is set when an external log sink is configured (e.g. type "cloudwatch"). CloudWatch* fields remain for backward compatibility.
	TaskLog *TaskLogSink    `json:"task_log,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// TaskStatusResponse is GET task-broker /v1/tasks/{id}.
type TaskStatusResponse struct {
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	FleetID         string     `json:"fleet_id"`
	CreatedAt       time.Time  `json:"created_at"`
	ClaimedAt       *time.Time `json:"claimed_at,omitempty"`
	LeaseUntil      *time.Time `json:"lease_until,omitempty"`
	RunnerID        string     `json:"runner_id,omitempty"` // set after claim (EC2: instance id from IMDS)
	ExecutionMode   string     `json:"execution_mode,omitempty"`
	DockerImage     string     `json:"docker_image,omitempty"`
	ExitCode        *int       `json:"exit_code,omitempty"`
	Error           string     `json:"error,omitempty"`
	CancelRequested bool       `json:"cancel_requested,omitempty"`
	// CloudWatchLogGroup and CloudWatchLogStream are set when task-broker is configured
	// with TASK_CLOUDWATCH_LOG_GROUP so clients can tail logs in AWS (runner must use the same group/prefix).
	CloudWatchLogGroup      string          `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream     string          `json:"cloudwatch_log_stream,omitempty"`
	TaskLog                 *TaskLogSink    `json:"task_log,omitempty"`
	ExecutionTimeoutSeconds *int            `json:"execution_timeout_seconds,omitempty"`
	Result                  json.RawMessage `json:"result,omitempty"`
}

// CancelTaskResponse is POST task-broker /v1/tasks/{id}/cancel.
type CancelTaskResponse struct {
	ID     string `json:"id"`
	State  string `json:"state"`  // already_terminal | canceled | cancel_requested
	Status string `json:"status"` // task status after the operation
}

// ListTasksResponse is GET task-broker /v1/tasks (non-terminal tasks only).
type ListTasksResponse struct {
	Tasks []TaskStatusResponse `json:"tasks"`
}

// BrokerGetTaskResponse is GET task-broker /v1/tasks/{id} (same shape as TaskStatusResponse).
type BrokerGetTaskResponse = TaskStatusResponse

// EffectiveRunMode returns the run mode for a create request (explicit or inferred).
func EffectiveRunMode(req *CreateTaskRequest) models.RunMode {
	if req == nil {
		return ""
	}
	if k := models.RunMode(strings.ToLower(strings.TrimSpace(req.RunMode))); k != "" {
		return k
	}
	return models.InferRunMode(NormalizeCommands(req.Commands), req.Command, strings.TrimSpace(req.Script))
}

// RunModeForTask returns the effective run mode on a claimed task payload.
func RunModeForTask(task *TaskPayload) models.RunMode {
	if task == nil {
		return ""
	}
	if k := models.RunMode(strings.ToLower(strings.TrimSpace(task.RunMode))); k != "" {
		return k
	}
	return models.InferRunMode(task.Commands, task.Command, strings.TrimSpace(task.Script))
}

// NormalizeCommands drops blank command-list entries and trims fields.
func NormalizeCommands(commands models.CommandList) models.CommandList {
	if len(commands) == 0 {
		return nil
	}
	out := make(models.CommandList, 0, len(commands))
	for _, spec := range commands {
		spec.Name = strings.TrimSpace(spec.Name)
		spec.Command = strings.TrimSpace(spec.Command)
		if spec.Command == "" {
			continue
		}
		out = append(out, spec)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeCommandLines trims and drops blank setup/command string lines.
func NormalizeCommandLines(commands []string) []string {
	var out []string
	for _, c := range commands {
		c = strings.TrimSpace(c)
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}

// CommandSpecsFromLines adapts plain shell lines into CommandSpec values (no names).
func CommandSpecsFromLines(lines []string) models.CommandList {
	normalized := NormalizeCommandLines(lines)
	if len(normalized) == 0 {
		return nil
	}
	out := make(models.CommandList, 0, len(normalized))
	for _, line := range normalized {
		out = append(out, models.CommandSpec{Command: line})
	}
	return out
}
