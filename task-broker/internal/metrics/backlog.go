package metrics

import (
	"context"
	"time"

	"github.com/superplane/runner/shared/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

type backlogKey struct {
	fleetID    string
	canvasName string
	nodeName   string
}

type backlogCounts struct {
	queued  int
	claimed int
}

// SampleTaskBacklog records queue backlog gauges for every fleet.
// Series are split by canvas_name / node_name when present on tasks.
func SampleTaskBacklog(ctx context.Context, st taskstore.Store, m *BrokerMetrics) error {
	return sampleTaskBacklogAt(ctx, st, m, time.Now().UTC())
}

func sampleTaskBacklogAt(ctx context.Context, st taskstore.Store, m *BrokerMetrics, now time.Time) error {
	fleets, err := st.ListFleets(ctx)
	if err != nil {
		return err
	}
	tasks, err := st.ListActiveTasks(ctx)
	if err != nil {
		return err
	}

	counts := make(map[backlogKey]*backlogCounts, len(fleets))
	for _, t := range tasks {
		key := backlogKey{
			fleetID:    t.FleetID,
			canvasName: t.Labels[models.LabelCanvasName],
			nodeName:   t.Labels[models.LabelNodeName],
		}
		c := counts[key]
		if c == nil {
			c = &backlogCounts{}
			counts[key] = c
		}
		switch t.Status {
		case models.StatusQueued:
			c.queued++
		case models.StatusClaimed:
			c.claimed++
		}
	}

	for key := range m.lastBacklogKeys {
		if _, ok := counts[key]; !ok {
			m.SetTaskBacklog(ctx, key.fleetID, key.canvasName, key.nodeName, 0, 0)
		}
	}

	next := make(map[backlogKey]struct{}, len(counts)+len(fleets))
	seenFleet := make(map[string]bool, len(fleets))
	for key, c := range counts {
		seenFleet[key.fleetID] = true
		m.SetTaskBacklog(ctx, key.fleetID, key.canvasName, key.nodeName, c.queued, c.claimed)
		next[key] = struct{}{}
	}
	for i := range fleets {
		id := fleets[i].ID
		if !seenFleet[id] {
			key := backlogKey{fleetID: id}
			m.SetTaskBacklog(ctx, id, "", "", 0, 0)
			next[key] = struct{}{}
		}
		oldestQueuedAt, err := st.OldestQueuedTaskCreatedAt(ctx, id)
		if err != nil {
			return err
		}
		m.SetOldestQueuedTaskAge(ctx, id, oldestQueuedAge(now, oldestQueuedAt))
	}
	m.lastBacklogKeys = next
	return nil
}

func oldestQueuedAge(now time.Time, oldestQueuedAt *time.Time) time.Duration {
	if oldestQueuedAt == nil {
		return 0
	}
	age := now.Sub(*oldestQueuedAt)
	if age < 0 {
		return 0
	}
	return age
}
