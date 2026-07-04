package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestReportDatabasePoolStats_RecordsConnectionGauges(t *testing.T) {
	database.TruncateTables()

	p := NewPeriodic(t.Context())
	p.reportDatabasePoolStats()

	stats, err := database.PoolStats()
	require.NoError(t, err)
	require.Greater(t, stats.MaxOpenConnections, 0)
}

func TestReportDatabasePoolStats_WaitDeltas(t *testing.T) {
	p := &Periodic{
		ctx:                  t.Context(),
		lastPoolWaitCount:    10,
		lastPoolWaitDuration: 2 * time.Second,
	}

	stats, err := database.PoolStats()
	require.NoError(t, err)

	p.lastPoolWaitCount = stats.WaitCount
	p.lastPoolWaitDuration = stats.WaitDuration

	// Simulate cumulative stats increasing between ticks.
	p.lastPoolWaitCount = stats.WaitCount - 3
	p.lastPoolWaitDuration = stats.WaitDuration - 150*time.Millisecond

	p.reportDatabasePoolStats()

	require.Equal(t, stats.WaitCount, p.lastPoolWaitCount)
	require.Equal(t, stats.WaitDuration, p.lastPoolWaitDuration)
}
