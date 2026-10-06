package models

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RunnerQueueWaitStarted  = "started"
	RunnerQueueWaitCanceled = "canceled"
	RunnerQueueWaitLost     = "lost"
)

// RunnerFleetLabels identify the fleet on a runner or task metric.
// OrganizationID is set only for organization-scoped fleets so two
// organizations can reuse one fleet slug without sharing a time series.
type RunnerFleetLabels struct {
	FleetSlug      string
	OrganizationID string
}

// RunnerMetrics receives runner and task observations after a durable
// state change. The telemetry package installs the recorder during metric
// setup. Model code cannot import telemetry because telemetry already
// imports this package.
type RunnerMetrics struct {
	StateOccupancy func(RunnerFleetLabels, string, time.Duration)
	TaskQueueWait  func(RunnerFleetLabels, string, time.Duration)
	TaskRun        func(RunnerFleetLabels, string, time.Duration)
}

var (
	runnerMetricsMu sync.RWMutex
	runnerMetrics   RunnerMetrics
)

func SetRunnerMetrics(next RunnerMetrics) RunnerMetrics {
	runnerMetricsMu.Lock()
	defer runnerMetricsMu.Unlock()
	previous := runnerMetrics
	runnerMetrics = next
	return previous
}

func currentRunnerMetrics() RunnerMetrics {
	runnerMetricsMu.RLock()
	defer runnerMetricsMu.RUnlock()
	return runnerMetrics
}

func runnerFleetLabels(tx *gorm.DB, fleetID uuid.UUID) (RunnerFleetLabels, bool) {
	var fleet RunnerFleet
	err := tx.Select("slug", "scope_type", "scope_id").
		Where("id = ?", fleetID).
		First(&fleet).
		Error
	if err != nil || fleet.Slug == "" {
		return RunnerFleetLabels{}, false
	}

	labels := RunnerFleetLabels{FleetSlug: fleet.Slug}
	if fleet.ScopeType == RunnerFleetScopeOrganization && fleet.ScopeID != nil {
		labels.OrganizationID = fleet.ScopeID.String()
	}
	return labels, true
}

// idleStartedAt is when this runner last entered idle.
// Connection heartbeats also update UpdatedAt, so that column is not the
// state-entry time. Idle begins at registration, or when the previous task
// finishes and a reusable runner returns to idle.
func (r *Runner) idleStartedAt(tx *gorm.DB) time.Time {
	var row struct {
		FinishedAt *time.Time
	}
	err := tx.Model(&RunnerTask{}).
		Select("MAX(finished_at) AS finished_at").
		Where("runner_id = ?", r.ID).
		Scan(&row).
		Error
	if err == nil && row.FinishedAt != nil && !row.FinishedAt.IsZero() {
		return *row.FinishedAt
	}
	if r.RegisteredAt != nil && !r.RegisteredAt.IsZero() {
		return *r.RegisteredAt
	}
	return r.CreatedAt
}

func recordRunnerStateOccupancy(tx *gorm.DB, fleetID uuid.UUID, state string, d time.Duration) {
	if d < 0 {
		return
	}
	labels, ok := runnerFleetLabels(tx, fleetID)
	if !ok {
		return
	}
	record := currentRunnerMetrics().StateOccupancy
	if record == nil {
		return
	}
	record(labels, state, d)
}

func recordRunnerTaskQueueWait(tx *gorm.DB, fleetID uuid.UUID, outcome string, queuedAt, now time.Time) {
	if queuedAt.IsZero() {
		return
	}
	d := now.Sub(queuedAt)
	if d < 0 {
		return
	}
	labels, ok := runnerFleetLabels(tx, fleetID)
	if !ok {
		return
	}
	record := currentRunnerMetrics().TaskQueueWait
	if record == nil {
		return
	}
	record(labels, outcome, d)
}

func recordRunnerTaskRun(tx *gorm.DB, fleetID uuid.UUID, state string, startedAt *time.Time, now time.Time) {
	if startedAt == nil || startedAt.IsZero() {
		return
	}
	d := now.Sub(*startedAt)
	if d < 0 {
		return
	}
	labels, ok := runnerFleetLabels(tx, fleetID)
	if !ok {
		return
	}
	record := currentRunnerMetrics().TaskRun
	if record == nil {
		return
	}
	record(labels, state, d)
}
