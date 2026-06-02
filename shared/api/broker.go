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
	ID     string   `json:"id"`
	Labels []string `json:"labels,omitempty"`
}

// FleetResponse describes a registered runner pool.
type FleetResponse struct {
	ID        string   `json:"id"`
	Labels    []string `json:"labels,omitempty"`
	CreatedAt int64    `json:"created_at_unix,omitempty"`
}

type FleetTaskCountsResponse struct {
	Queued  int `json:"queued"`
	Claimed int `json:"claimed"`
}
