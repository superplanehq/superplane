package models

import "time"

// TaskStatus is persisted for queue and outcome state.
type TaskStatus string

const (
	StatusQueued    TaskStatus = "queued"
	StatusClaimed   TaskStatus = "claimed"
	StatusSucceeded TaskStatus = "succeeded"
	StatusFailed    TaskStatus = "failed"
	StatusCanceled  TaskStatus = "canceled"
)

// ExecutionMode selects how the runner executes the command.
type ExecutionMode string

const (
	ExecutionHost   ExecutionMode = "host"
	ExecutionDocker ExecutionMode = "docker"
)

// Task is work submitted to the task queue.
type Task struct {
	ID      string
	FleetID string
	// RunMode selects command_list, argv, or a script mode. Empty means infer from fields.
	RunMode RunMode
	// Script is user code (main() or main(payload) entrypoint) when RunMode is a script mode.
	Script string
	// MessageChainJSON is the SuperPlane $ message chain passed to script tasks.
	MessageChainJSON string
	// Command is argv for a single process when Commands is empty (legacy).
	Command []string
	// Commands are shell directives when non-empty (Bash+PTY sourcing per line on workers).
	Commands []string
	// SetupCommands are optional shell directives run before script tasks.
	SetupCommands []string
	// Environment is a task-scoped process environment sent only to runners.
	Environment   []EnvironmentVariable
	WebhookURL    string
	Status        TaskStatus
	CreatedAt     time.Time
	ClaimedAt     *time.Time
	LeaseUntil    *time.Time
	RunnerID      string
	ExecutionMode ExecutionMode
	DockerImage   string
	// ExecutionTimeoutSeconds when non-nil is the per-task wall-clock limit in seconds.
	ExecutionTimeoutSeconds *int
	ExitCode                *int
	Output                  string
	// ResultJSON is optional structured JSON from the runner (SUPERPLANE_RESULT_FILE).
	ResultJSON   string
	ErrorMessage string
	// InfraRetryCount counts automatic retries after runner infrastructure failures.
	InfraRetryCount int
	// CancelRequested is true while the task is claimed and a caller asked to stop it (persisted).
	CancelRequested bool
}

// EnvironmentVariable is one task-scoped environment variable.
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
