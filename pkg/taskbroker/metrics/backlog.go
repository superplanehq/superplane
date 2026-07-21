package metrics

import (
	"context"

	taskstore "github.com/superplanehq/superplane/pkg/taskbroker/store"
)

// SampleTaskBacklog records tasks.queued and tasks.claimed gauges for every fleet.
func SampleTaskBacklog(ctx context.Context, st taskstore.Store, m *BrokerMetrics) error {
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
	}
	return nil
}
