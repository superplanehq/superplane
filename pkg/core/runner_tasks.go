package core

import (
	"encoding/json"
	"time"
)

const (
	RunnerTaskBackendLegacy     = "legacy"
	RunnerTaskBackendIntegrated = "integrated"
)

// RunnerTaskContext gives runner components access to SuperPlane-owned tasks
// without coupling component implementations to GORM or encryption details.
// Implementations are bound to the current worker transaction.
type RunnerTaskContext interface {
	IntegratedBackendEnabled() (bool, error)
	Create(id, fleetID string, payload []byte) error
	Find(id string) (*RunnerTask, error)
	RequestCancel(id string) error
}

// RunnerTask is the component-facing view of a durable integrated task.
type RunnerTask struct {
	ID           string
	State        string
	Result       json.RawMessage
	ExitCode     *int
	ErrorMessage string
	ReservedAt   *time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}
