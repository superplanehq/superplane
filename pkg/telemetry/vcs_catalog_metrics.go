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

// A GitHub installation token with less than this share of its hourly
// request limit left is close to being rate limited.
const githubRateLimitLowRatio = 0.1

var (
	vcsCatalogDueSyncJobsGauge        metric.Int64Gauge
	vcsCatalogOldestDueSyncJobGauge   metric.Float64Gauge
	vcsCatalogRepositorySyncsCounter  metric.Int64Counter
	vcsCatalogRepositorySyncHistogram metric.Float64Histogram
	vcsCatalogGitHubRateLimitCounter  metric.Int64Counter
)

func registerVCSCatalogMetrics() error {
	var err error
	vcsCatalogDueSyncJobsGauge, err = meter.Int64Gauge("vcs_catalog.sync_jobs.due",
		metric.WithDescription("Repository collaborator sync jobs that are due and not claimed"), metric.WithUnit("1"))
	if err != nil {
		return err
	}
	vcsCatalogOldestDueSyncJobGauge, err = meter.Float64Gauge("vcs_catalog.sync_jobs.oldest_due.age.seconds",
		metric.WithDescription("Time since the oldest unclaimed repository collaborator sync job became due"), metric.WithUnit("s"))
	if err != nil {
		return err
	}
	vcsCatalogRepositorySyncsCounter, err = meter.Int64Counter("vcs_catalog.repository_syncs.total",
		metric.WithDescription("Repository collaborator sync outcomes"), metric.WithUnit("1"))
	if err != nil {
		return err
	}
	vcsCatalogRepositorySyncHistogram, err = meter.Float64Histogram("vcs_catalog.repository_sync.duration.seconds",
		metric.WithDescription("Duration of repository collaborator syncs"), metric.WithUnit("s"))
	if err != nil {
		return err
	}
	vcsCatalogGitHubRateLimitCounter, err = meter.Int64Counter("vcs_catalog.github.rate_limit.responses.total",
		metric.WithDescription("GitHub catalog API responses by the state of the installation rate limit"), metric.WithUnit("1"))
	return err
}

func RecordVCSCatalogRepositorySync(ctx context.Context, provider string, d time.Duration, outcome string) {
	if !metricsReady.Load() {
		return
	}
	vcsCatalogRepositorySyncsCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("outcome", outcome),
	))
	vcsCatalogRepositorySyncHistogram.Record(ctx, d.Seconds(), metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("outcome", outcome),
	))
}

// RecordGitHubCatalogRateLimit records whether a GitHub response came from an
// installation token that is close to its rate limit. Rate limits apply to each
// installation, so a counter of low responses stays useful across thousands of
// installations without one series for each installation.
func RecordGitHubCatalogRateLimit(ctx context.Context, remaining, limit int) {
	if !metricsReady.Load() || limit <= 0 {
		return
	}
	state := "ok"
	if float64(remaining) < float64(limit)*githubRateLimitLowRatio {
		state = "low"
	}
	vcsCatalogGitHubRateLimitCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("state", state)))
}

func (p *Periodic) reportVCSCatalogSyncQueue() {
	if !metricsReady.Load() {
		return
	}
	now := time.Now()
	stats, err := models.ListVCSProviderRepositorySyncQueueStats(database.DB(p.ctx), now)
	if err != nil {
		log.WithError(err).Error("failed to report VCS catalog sync queue")
		return
	}

	// Record zero for an empty queue, so the gauge does not keep its last value.
	type queueKey struct {
		provider string
		priority models.VCSProviderRepositorySyncPriority
	}
	reported := map[queueKey]models.VCSProviderRepositorySyncQueueStat{
		{models.ProviderGitHub, models.VCSProviderRepositorySyncPriorityBackground}:  {},
		{models.ProviderGitHub, models.VCSProviderRepositorySyncPriorityInteractive}: {},
	}
	for _, stat := range stats {
		reported[queueKey{stat.Provider, stat.Priority}] = stat
	}

	for key, stat := range reported {
		attrs := metric.WithAttributes(
			attribute.String("provider", key.provider),
			attribute.String("priority", vcsCatalogSyncPriorityName(key.priority)),
		)
		vcsCatalogDueSyncJobsGauge.Record(p.ctx, stat.Due, attrs)
		age := 0.0
		if stat.Due > 0 {
			age = max(now.Sub(stat.OldestRunAt).Seconds(), 0)
		}
		vcsCatalogOldestDueSyncJobGauge.Record(p.ctx, age, attrs)
	}
}

func vcsCatalogSyncPriorityName(priority models.VCSProviderRepositorySyncPriority) string {
	if priority >= models.VCSProviderRepositorySyncPriorityInteractive {
		return "interactive"
	}
	return "background"
}
