package models

import "encoding/json"

// FleetSyncResponse is returned by POST /api/v1/runner-fleets/sync.
type FleetSyncResponse struct {
	Continue bool            `json:"continue"`
	Job      *FleetBridgeJob `json:"job,omitempty"`
}

// FleetBridgeJob is a queued job for fleet-manager to run locally.
type FleetBridgeJob struct {
	ID   string  `json:"id"`
	Spec JobSpec `json:"spec"`
}

// FleetCompleteRequest is POST /api/v1/runner-fleets/tasks/{id}/complete from fleet-manager.
type FleetCompleteRequest struct {
	ExitCode int             `json:"exit_code"`
	Output   string          `json:"output"`
	Error    string          `json:"error,omitempty"`
	Canceled bool            `json:"canceled,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`
	TaskLog  *TaskLogSink    `json:"task_log,omitempty"`
}
