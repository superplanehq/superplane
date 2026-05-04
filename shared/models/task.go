package models

import "time"

// TaskStatus is persisted for queue and outcome state.
type TaskStatus string

const (
	StatusQueued    TaskStatus = "queued"
	StatusClaimed   TaskStatus = "claimed"
	StatusSucceeded TaskStatus = "succeeded"
	StatusFailed    TaskStatus = "failed"
)

// ExecutionMode selects how the runner executes the command.
type ExecutionMode string

const (
	ExecutionHost   ExecutionMode = "host"
	ExecutionDocker ExecutionMode = "docker"
)

// Task is work submitted to fleet-manager.
type Task struct {
	ID            string
	Command       []string
	WebhookURL    string
	Status        TaskStatus
	CreatedAt     time.Time
	ClaimedAt     *time.Time
	LeaseUntil    *time.Time
	RunnerID      string
	ExecutionMode ExecutionMode
	DockerImage   string
	ExitCode      *int
	Output        string
	ErrorMessage  string
}
