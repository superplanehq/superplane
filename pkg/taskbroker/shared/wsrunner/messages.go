// Package wsrunner defines JSON message types for the fleet-manager runner WebSocket
// stream (GET /v1/runners/stream).
//
// Ordering: after server sends type "task", it may send type "cancel" (same task_id) while
// the client runs the task; the client then sends type "complete" and reads type "ack" or "error".
package wsrunner

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
)

// Message type discriminator values (JSON field "type").
const (
	TypeHello    = "hello"
	TypeTask     = "task"
	TypeCancel   = "cancel"
	TypeComplete = "complete"
	TypeAck      = "ack"
	TypeError    = "error"
)

// Hello is the first client message after connect. Fields match api.ClaimTaskRequest JSON tags.
type Hello struct {
	Type              string `json:"type"`
	RunnerID          string `json:"runner_id"`
	FleetID           string `json:"fleet_id"`
	LeaseSeconds      int    `json:"lease_seconds"`
	LaunchRequestedAt int64  `json:"launch_requested_at,omitempty"`
	// OneShot instructs the broker to close the stream after delivering exactly one task.
	// Set by runners configured with ExitAfterEachTask=true so the broker never races to
	// claim a second task before the runner's WS close is observed.
	OneShot bool `json:"one_shot,omitempty"`
}

// Task is server -> client after a successful claim.
type Task struct {
	Type string           `json:"type"`
	Task *api.TaskPayload `json:"task"`
}

// Cancel is server -> client: caller requested stop for the in-flight task.
type Cancel struct {
	Type   string `json:"type"`
	TaskID string `json:"task_id"`
}

// Complete is client -> server to finish a claimed task.
type Complete struct {
	Type        string          `json:"type"`
	TaskID      string          `json:"task_id"`
	RunnerID    string          `json:"runner_id"`
	ExitCode    int             `json:"exit_code"`
	Error       string          `json:"error,omitempty"`
	FailureKind string          `json:"failure_kind,omitempty"`
	Canceled    bool            `json:"canceled,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
}

// Ack is server -> client after successful complete (HTTP 204 equivalent).
type Ack struct {
	Type string `json:"type"`
	OK   bool   `json:"ok"`
}

// Error is server -> client when complete fails or protocol is invalid.
type Error struct {
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}
