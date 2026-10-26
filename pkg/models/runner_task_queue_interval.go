package models

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RunnerTaskQueueInterval struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	StartedAt time.Time
	EndedAt   *time.Time
}

// OpenQueueInterval is idempotent with the database queue transition trigger.
func (t *RunnerTask) OpenQueueInterval(tx *gorm.DB, now time.Time) error {
	return tx.Exec(`INSERT INTO runner_task_queue_intervals(task_id, started_at)
		SELECT id, ? FROM runner_tasks WHERE id = ? AND backend = ? AND state = ? AND runner_id IS NULL
		ON CONFLICT DO NOTHING`, now, t.ID, RunnerTaskBackendIntegrated, RunnerTaskStateQueued).Error
}

func (t *RunnerTask) CloseQueueInterval(tx *gorm.DB, now time.Time) error {
	return tx.Model(&RunnerTaskQueueInterval{}).Where("task_id = ? AND ended_at IS NULL", t.ID).
		Update("ended_at", gorm.Expr("GREATEST(started_at, ?)", now)).Error
}

// RunnerQueueWait measures the union in a child run and all its descendants.
// Parallel waits pause the parent only once.
func (r *CanvasRun) RunnerQueueWait(tx *gorm.DB, from, until time.Time) (time.Duration, bool, error) {
	var intervals []RunnerTaskQueueInterval
	err := tx.Raw(`WITH RECURSIVE descendants AS (
		SELECT id FROM workflow_runs WHERE id = ?
		UNION
		SELECT r.id FROM workflow_runs r JOIN descendants d ON r.parent_run_id = d.id
	)
	SELECT DISTINCT i.* FROM runner_task_queue_intervals i
	JOIN workflow_node_execution_kvs kv ON kv.key = 'task_id' AND kv.value = i.task_id::text
	JOIN workflow_node_executions e ON e.id = kv.execution_id AND e.workflow_id = kv.workflow_id AND e.node_id = kv.node_id
	JOIN descendants d ON d.id = e.run_id
	WHERE i.started_at < ? AND (i.ended_at IS NULL OR i.ended_at > ?)
	ORDER BY i.started_at, i.id`, r.ID, until, from).Scan(&intervals).Error
	if err != nil {
		return 0, false, err
	}
	duration, open := unionQueueWait(intervals, from, until)
	return duration, open, nil
}

func unionQueueWait(intervals []RunnerTaskQueueInterval, from, until time.Time) (time.Duration, bool) {
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].StartedAt.Before(intervals[j].StartedAt) })
	var total time.Duration
	var end time.Time
	open := false
	for _, interval := range intervals {
		start := maxTime(interval.StartedAt, from)
		stop := until
		if interval.EndedAt != nil {
			stop = minTime(*interval.EndedAt, until)
		}
		if !stop.After(start) {
			continue
		}
		open = open || interval.EndedAt == nil
		start = maxTime(start, end)
		if stop.After(start) {
			total += stop.Sub(start)
			end = stop
		}
	}
	return total, open
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
