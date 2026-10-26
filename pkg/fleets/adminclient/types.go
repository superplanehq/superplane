package adminclient

import "time"

const (
	RunnerStatePending    = "pending"
	RunnerStateIdle       = "idle"
	RunnerStateBusy       = "busy"
	RunnerStateTerminated = "terminated"
)

type FleetSpec struct {
	OperatingSystem string   `json:"operatingSystem"`
	Architecture    string   `json:"architecture"`
	CPUMillicores   int32    `json:"cpuMillicores"`
	MemoryMB        int32    `json:"memoryMb"`
	DiskGB          int32    `json:"diskGb"`
	Capabilities    []string `json:"capabilities"`
}

type Fleet struct {
	ID                        string     `json:"id"`
	Enabled                   bool       `json:"enabled"`
	Spec                      FleetSpec  `json:"spec"`
	RunnerVersion             string     `json:"runnerVersion"`
	CreatedAt                 time.Time  `json:"createdAt"`
	UpdatedAt                 time.Time  `json:"updatedAt"`
	ProvisioningCooldownUntil *time.Time `json:"provisioningCooldownUntil,omitempty"`
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
	ReservedTaskID    *string    `json:"reservedTaskId,omitempty"`
}

type Capacity struct {
	RunnableTasks             int64      `json:"runnableTasks,string"`
	PendingRunners            int64      `json:"pendingRunners,string"`
	IdleRunners               int64      `json:"idleRunners,string"`
	BusyRunners               int64      `json:"busyRunners,string"`
	TerminatedRunners         int64      `json:"terminatedRunners,string"`
	Generation                string     `json:"generation"`
	UnassignedRunners         int64      `json:"unassignedRunners,string"`
	ProvisioningCooldownUntil *time.Time `json:"provisioningCooldownUntil,omitempty"`
}

type CreateRunnerRequest struct {
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	Ephemeral      bool   `json:"ephemeral"`
	TaskID         string `json:"taskId,omitempty"`
	WarmCapacity   *int32 `json:"warmCapacity,omitempty"`
	MaxCapacity    *int32 `json:"maxCapacity,omitempty"`
}

type CreateRunnerResponse struct {
	Runner                Runner    `json:"runner"`
	RegistrationToken     string    `json:"registrationToken"`
	RegistrationExpiresAt time.Time `json:"registrationExpiresAt"`
	RunnerAPIURL          string    `json:"runnerApiUrl"`
	DisplayName           string    `json:"displayName"`
}

type CreateFleetRequest struct {
	FleetID       string    `json:"fleetId"`
	RunnerVersion string    `json:"runnerVersion"`
	Spec          FleetSpec `json:"spec"`
}

type UpdateFleetRequest struct {
	RunnerVersion string    `json:"runnerVersion"`
	Spec          FleetSpec `json:"spec"`
}

type Task struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	State          string    `json:"state"`
	QueuedAt       time.Time `json:"queuedAt"`
}

type DeleteRunnerOptions struct {
	RequireUnassigned bool
	Reason            string
}
