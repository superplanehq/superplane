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

// Alert rules use these metrics. Alert when sentry.webhooks.inbox.blocked
// is above zero, and when sentry.webhooks.inbox.oldest_unfinished.age
// is above 300 seconds.
var (
	sentryWebhookAcceptanceHistogram metric.Float64Histogram
	sentryWebhookProcessingHistogram metric.Float64Histogram
	sentryWebhookAttemptsCounter     metric.Int64Counter
	sentryWebhookUnfinishedGauge     metric.Int64Gauge
	sentryWebhookBlockedGauge        metric.Int64Gauge
	sentryWebhookOldestUnfinishedAge metric.Float64Gauge
)

func registerSentryWebhookMetrics() error {
	var err error
	sentryWebhookAcceptanceHistogram, err = meter.Float64Histogram("sentry.webhooks.acceptance.duration.seconds",
		metric.WithDescription("Time from a received Sentry webhook to the answer, including the durable commit"), metric.WithUnit("s"))
	if err != nil {
		return err
	}
	sentryWebhookProcessingHistogram, err = meter.Float64Histogram("sentry.webhooks.processing.duration.seconds",
		metric.WithDescription("Duration of one Sentry webhook inbox attempt"), metric.WithUnit("s"))
	if err != nil {
		return err
	}
	sentryWebhookAttemptsCounter, err = meter.Int64Counter("sentry.webhooks.attempts",
		metric.WithDescription("Sentry webhook inbox attempts by recorded state; retrying and blocked count retries"), metric.WithUnit("1"))
	if err != nil {
		return err
	}
	sentryWebhookUnfinishedGauge, err = meter.Int64Gauge("sentry.webhooks.inbox.unfinished",
		metric.WithDescription("Sentry webhook deliveries that are not finished"), metric.WithUnit("1"))
	if err != nil {
		return err
	}
	sentryWebhookBlockedGauge, err = meter.Int64Gauge("sentry.webhooks.inbox.blocked",
		metric.WithDescription("Sentry webhook deliveries that wait for an operator or user action"), metric.WithUnit("1"))
	if err != nil {
		return err
	}
	sentryWebhookOldestUnfinishedAge, err = meter.Float64Gauge("sentry.webhooks.inbox.oldest_unfinished.age",
		metric.WithDescription("Age of the oldest unfinished Sentry webhook delivery"), metric.WithUnit("s"))
	return err
}

// RecordSentryWebhookAcceptance records how long Sentry waited for an answer.
func RecordSentryWebhookAcceptance(ctx context.Context, d time.Duration, source string, status int) {
	if !metricsReady.Load() {
		return
	}
	sentryWebhookAcceptanceHistogram.Record(ctx, d.Seconds(), metric.WithAttributes(
		attribute.String("source", source),
		attribute.Int("status", status),
	))
}

// RecordSentryWebhookAttempt records one inbox attempt and its state.
func RecordSentryWebhookAttempt(ctx context.Context, d time.Duration, state string) {
	if !metricsReady.Load() {
		return
	}
	attributes := metric.WithAttributes(attribute.String("state", state))
	sentryWebhookProcessingHistogram.Record(ctx, d.Seconds(), attributes)
	sentryWebhookAttemptsCounter.Add(ctx, 1, attributes)
}

func (p *Periodic) reportSentryWebhookInbox() {
	if !metricsReady.Load() {
		return
	}
	stats, err := models.FindSentryWebhookInboxStats(database.DB(p.ctx), time.Now())
	if err != nil {
		log.WithError(err).Error("failed to report Sentry webhook inbox metrics")
		return
	}
	sentryWebhookUnfinishedGauge.Record(p.ctx, stats.Unfinished)
	sentryWebhookBlockedGauge.Record(p.ctx, stats.Blocked)
	sentryWebhookOldestUnfinishedAge.Record(p.ctx, stats.OldestAge.Seconds())
}
