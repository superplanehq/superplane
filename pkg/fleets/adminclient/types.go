package adminclient

import "time"

const (
	RunnerStatePending    = "pending"
	RunnerStateIdle       = "idle"
	RunnerStateBusy       = "busy"
	RunnerStateTerminated = "terminated"

	TaskStateQueued   = "queued"
	TaskStateReserved = "reserved"
)

type FleetSpec struct {
	OperatingSystem            string   `json:"operatingSystem"`
	Architecture               string   `json:"architecture"`
	CPUMillicores              int32    `json:"cpuMillicores"`
	MemoryMB                   int32    `json:"memoryMb"`
	DiskGB                     int32    `json:"diskGb"`
	Capabilities               []string `json:"capabilities"`
	MaxExecutionTimeoutSeconds int32    `json:"maxExecutionTimeoutSeconds"`
}

type Fleet struct {
	ID            string    `json:"id"`
	Enabled       bool      `json:"enabled"`
	Spec          FleetSpec `json:"spec"`
	RunnerVersion string    `json:"runnerVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Runner struct {
	ID                string     `json:"id"`
	FleetID           string     `json:"fleetId"`
	State             string     `json:"state"`
	RunnerVersion     string     `json:"runnerVersion"`
	RegisteredAt      *time.Time `json:"registeredAt,omitempty"`
	LastSeenAt        *time.Time `json:"lastSeenAt,omitempty"`
	TerminationReason *string    `json:"terminationReason,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	TerminatedAt      *time.Time `json:"terminatedAt,omitempty"`
	Ephemeral         bool       `json:"ephemeral"`
}

type Task struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	FleetID        string     `json:"fleetId"`
	RunnerID       *string    `json:"runnerId,omitempty"`
	State          string     `json:"state"`
	QueuedAt       time.Time  `json:"queuedAt"`
	ReservedAt     *time.Time `json:"reservedAt,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
}

type Capacity struct {
	RunnableTasks     int64  `json:"runnableTasks,string"`
	PendingRunners    int64  `json:"pendingRunners,string"`
	IdleRunners       int64  `json:"idleRunners,string"`
	BusyRunners       int64  `json:"busyRunners,string"`
	TerminatedRunners int64  `json:"terminatedRunners,string"`
	Generation        string `json:"generation"`
}

type CreateRunnerRequest struct {
	TaskID         *string `json:"taskId,omitempty"`
	IdempotencyKey string  `json:"idempotencyKey,omitempty"`
	Ephemeral      bool    `json:"ephemeral"`
}

type CreateRunnerResponse struct {
	Runner                Runner    `json:"runner"`
	RegistrationToken     string    `json:"registrationToken"`
	RegistrationExpiresAt time.Time `json:"registrationExpiresAt"`
	RunnerAPIURL          string    `json:"runnerApiUrl"`
	DisplayName           string    `json:"displayName"`
}
