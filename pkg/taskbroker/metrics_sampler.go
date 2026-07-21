package taskbroker

import (
	"context"
	"log/slog"
	"time"

	brokermetrics "github.com/superplanehq/superplane/pkg/taskbroker/metrics"
	"github.com/superplanehq/superplane/pkg/taskbroker/store"
)

func runMetricsSampler(ctx context.Context, log *slog.Logger, st store.Store, m *brokermetrics.BrokerMetrics, interval time.Duration) {
	if interval <= 0 || m == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sample := func() {
		if err := brokermetrics.SampleTaskBacklog(context.Background(), st, m); err != nil && log != nil {
			log.Warn("sample task backlog metrics", slog.Any("err", err))
		}
	}
	sample()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sample()
		}
	}
}
