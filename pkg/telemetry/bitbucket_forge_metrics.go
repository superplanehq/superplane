package telemetry

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Allow three scheduled five-minute deliveries to be missed before reporting
// a stale installation. This is independent of the cached token's lifetime.
const bitbucketForgeDeliveryGracePeriod = 15 * time.Minute

var (
	bitbucketForgeDeliveriesCounter       metric.Int64Counter
	bitbucketForgeStaleInstallationsGauge metric.Int64Gauge
)

func registerBitbucketForgeMetrics() error {
	var err error
	bitbucketForgeDeliveriesCounter, err = meter.Int64Counter("bitbucket.forge.deliveries",
		metric.WithDescription("Bitbucket Forge delivery outcomes"), metric.WithUnit("1"))
	if err != nil {
		return err
	}
	bitbucketForgeStaleInstallationsGauge, err = meter.Int64Gauge("bitbucket.forge.installations.stale",
		metric.WithDescription("Installed Bitbucket workspaces with missing credentials or no delivery for fifteen minutes"), metric.WithUnit("1"))
	return err
}

func RecordBitbucketForgeDelivery(ctx context.Context, outcome string) {
	if !metricsReady.Load() {
		return
	}
	bitbucketForgeDeliveriesCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}

func (p *Periodic) reportBitbucketForgeDeliveries() {
	if !metricsReady.Load() {
		return
	}
	now := time.Now()
	count, err := models.CountUnhealthyBitbucketForgeInstallations(database.DB(p.ctx), now, now.Add(-bitbucketForgeDeliveryGracePeriod))
	if err != nil {
		log.WithError(err).Error("failed to report Bitbucket Forge delivery health")
		return
	}
	bitbucketForgeStaleInstallationsGauge.Record(p.ctx, count)
}
