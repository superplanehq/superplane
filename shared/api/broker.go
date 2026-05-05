package api

// BrokerCreateTaskRequest is POST task-broker /v1/tasks.
// Embedding carries command(s), webhook_url, execution_mode, etc. for upstream fleet-manager.
//
// Exactly one routing field must be set: FleetID selects a fleet by id,
// FleetLabels selects a fleet that has all listed labels (smallest FleetID wins).
type BrokerCreateTaskRequest struct {
	CreateTaskRequest
	FleetID     string   `json:"fleet_id,omitempty"`
	FleetLabels []string `json:"fleet_labels,omitempty"`
}

// BrokerCreateTaskResponse returns the broker-scoped task id (used in forwarded webhooks as task_id).
type BrokerCreateTaskResponse struct {
	ID string `json:"id"`
}

// RegisterFleetRequest is POST task-broker /v1/fleets.
type RegisterFleetRequest struct {
	ID          string   `json:"id"`
	BaseURL     string   `json:"base_url"`
	AuthToken   string   `json:"auth_token,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// FleetResponse describes a registered downstream fleet-manager.
type FleetResponse struct {
	ID        string   `json:"id"`
	BaseURL   string   `json:"base_url"`
	Labels    []string `json:"labels,omitempty"`
	CreatedAt int64    `json:"created_at_unix,omitempty"`
}

// Caller completion notifications use WebhookPayload; when the task ran via a fleet-manager
// proxied through task-broker, task_id is the broker scope id and fleet_task_id is set.
