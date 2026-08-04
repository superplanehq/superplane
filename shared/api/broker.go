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
	ID                         string `json:"id"`
	Provisioner                string `json:"provisioner,omitempty"`
	Arch                       string `json:"arch,omitempty"`
	Size                       string `json:"size,omitempty"`
	DispatchTarget             string `json:"dispatch_target,omitempty"`
	MaxExecutionTimeoutSeconds *int   `json:"max_execution_timeout_seconds,omitempty"`
	SupportsDocker             *bool  `json:"supports_docker,omitempty"`
}

// FleetResponse describes a registered runner pool.
type FleetResponse struct {
	ID                         string `json:"id"`
	Provisioner                string `json:"provisioner,omitempty"`
	Arch                       string `json:"arch,omitempty"`
	Size                       string `json:"size,omitempty"`
	CreatedAt                  int64  `json:"created_at_unix,omitempty"`
	DispatchTarget             string `json:"dispatch_target,omitempty"`
	MaxExecutionTimeoutSeconds *int   `json:"max_execution_timeout_seconds,omitempty"`
	SupportsDocker             *bool  `json:"supports_docker,omitempty"`
}

type FleetTaskCountsResponse struct {
	Queued           int      `json:"queued"`
	Claimed          int      `json:"claimed"`
	ClaimedRunnerIDs []string `json:"claimed_runner_ids,omitempty"`
}

type DrainRunnersRequest struct {
	FleetID              string      `json:"fleet_id"`
	RunnerIDs            []string    `json:"runner_ids"`
	Reason               DrainReason `json:"reason,omitempty"`
	TerminationConfirmed bool        `json:"termination_confirmed,omitempty"`
}

type DrainReason string

const (
	DrainReasonScaleDown DrainReason = "scale_down"
	DrainReasonUnhealthy DrainReason = "unhealthy"
)

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
	Runners        []DrainRunnerStatus  `json:"runners"`
	RecoveredTasks []RunnerTaskRecovery `json:"recovered_tasks,omitempty"`
}

type RunnerTaskRecoveryState string

const (
	RunnerTaskRecoveryStateFailed   RunnerTaskRecoveryState = "failed"
	RunnerTaskRecoveryStateCanceled RunnerTaskRecoveryState = "canceled"
)

type RunnerTaskRecovery struct {
	RunnerID string                  `json:"runner_id"`
	TaskID   string                  `json:"task_id"`
	State    RunnerTaskRecoveryState `json:"state"`
}
