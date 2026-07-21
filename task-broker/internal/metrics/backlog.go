package metrics

import (
	"context"
	"time"

	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

// SampleTaskBacklog records queue backlog gauges for every fleet.
func SampleTaskBacklog(ctx context.Context, st taskstore.Store, m *BrokerMetrics) error {
	return sampleTaskBacklogAt(ctx, st, m, time.Now().UTC())
}

func sampleTaskBacklogAt(ctx context.Context, st taskstore.Store, m *BrokerMetrics, now time.Time) error {
	fleets, err := st.ListFleets(ctx)
	if err != nil {
		return err
	}
	for i := range fleets {
		queued, claimed, err := st.CountTasksByFleet(ctx, fleets[i].ID)
		if err != nil {
			return err
		}
		m.SetTaskBacklog(ctx, fleets[i].ID, queued, claimed)
		oldestQueuedAt, err := st.OldestQueuedTaskCreatedAt(ctx, fleets[i].ID)
		if err != nil {
			return err
		}
		m.SetOldestQueuedTaskAge(ctx, fleets[i].ID, oldestQueuedAge(now, oldestQueuedAt))
	}
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
