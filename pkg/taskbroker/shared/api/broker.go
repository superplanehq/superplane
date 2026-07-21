package api

// BrokerCreateTaskRequest is POST task-broker /v1/tasks.
type BrokerCreateTaskRequest struct {
	CreateTaskRequest
	FleetID string `json:"fleet_id,omitempty"`
}

// BrokerCreateTaskResponse returns the task id.
type BrokerCreateTaskResponse struct {
	ID string `json:"id"`
}

// RegisterFleetRequest is POST task-broker /v1/fleets.
type RegisterFleetRequest struct {
	ID          string `json:"id"`
	Provisioner string `json:"provisioner,omitempty"`
	Arch        string `json:"arch,omitempty"`
	Size        string `json:"size,omitempty"`
}

// FleetResponse describes a registered runner pool.
type FleetResponse struct {
	ID          string `json:"id"`
	Provisioner string `json:"provisioner,omitempty"`
	Arch        string `json:"arch,omitempty"`
	Size        string `json:"size,omitempty"`
	CreatedAt   int64  `json:"created_at_unix,omitempty"`
}

type FleetTaskCountsResponse struct {
	Queued           int      `json:"queued"`
	Claimed          int      `json:"claimed"`
	ClaimedRunnerIDs []string `json:"claimed_runner_ids,omitempty"`
}

type DrainRunnersRequest struct {
	FleetID   string   `json:"fleet_id"`
	RunnerIDs []string `json:"runner_ids"`
}

type DrainRunnerState string

const (
	DrainRunnerStateDrained DrainRunnerState = "drained"
	DrainRunnerStateBusy    DrainRunnerState = "busy"
)

type DrainRunnerStatus struct {
	RunnerID     string           `json:"runner_id"`
	State        DrainRunnerState `json:"state"`
	ActiveTaskID string           `json:"active_task_id,omitempty"`
}

type DrainRunnersResponse struct {
	Runners []DrainRunnerStatus `json:"runners"`
}
