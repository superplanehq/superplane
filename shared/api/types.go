package api

import "github.com/superplane/runner/shared/models"

// CreateTaskRequest is POST /v1/tasks.
type CreateTaskRequest struct {
	// Command is argv for one process. Omit when using Commands.
	Command []string `json:"command,omitempty"`
	// Commands are lines of one shell script: joined with newlines and run as a single
	// sh -c so export, cd, and shell state persist between lines. Omit when using Command.
	Commands      []string `json:"commands,omitempty"`
	WebhookURL    string   `json:"webhook_url"`
	ExecutionMode string   `json:"execution_mode"` // "host" | "docker"
	DockerImage   string   `json:"docker_image,omitempty"`
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
	Commands      []string `json:"commands,omitempty"` // one script, lines joined with newlines; see CreateTaskRequest
	ExecutionMode string   `json:"execution_mode"`
	DockerImage   string   `json:"docker_image,omitempty"`
}

// CompleteTaskRequest is POST /v1/tasks/{id}/complete.
type CompleteTaskRequest struct {
	RunnerID string `json:"runner_id"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// TaskPayloadFrom maps models.Task to TaskPayload.
func TaskPayloadFrom(t *models.Task) *TaskPayload {
	return &TaskPayload{
		ID:            t.ID,
		Command:       t.Command,
		Commands:      t.Commands,
		ExecutionMode: string(t.ExecutionMode),
		DockerImage:   t.DockerImage,
	}
}

// WebhookPayload is POSTed to the caller webhook URL on terminal status.
type WebhookPayload struct {
	TaskID      string `json:"task_id"`
	FleetTaskID string `json:"fleet_task_id,omitempty"` // when task-broker forwards, fleet-managed id
	Status      string `json:"status"`
	ExitCode    int    `json:"exit_code"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
}
