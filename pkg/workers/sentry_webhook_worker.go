package workers

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/sentryinbox"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

const (
	sentryWebhookPollInterval  = time.Second
	sentryWebhookBatchSize     = 25
	sentryWebhookConcurrency   = 5
	sentryWebhookLeaseDuration = 5 * time.Minute
	sentryWebhookLeaseRenewal  = time.Minute
)

// SentryWebhookWorker processes the durable Sentry webhook inbox. It uses
// the integration request lease pattern: a short claim transaction with
// SKIP LOCKED, external calls outside transactions, and short completion
// transactions that confirm the lease first.
type SentryWebhookWorker struct {
	processor     *sentryinbox.Processor
	owner         uuid.UUID
	pollInterval  time.Duration
	batchSize     int
	concurrency   int
	leaseDuration time.Duration
	leaseRenewal  time.Duration
	logger        *log.Entry
}

func NewSentryWebhookWorker(encryptor crypto.Encryptor, registry *registry.Registry) *SentryWebhookWorker {
	return NewSentryWebhookWorkerWithProcessor(sentryinbox.NewProcessor(encryptor, registry))
}

func NewSentryWebhookWorkerWithProcessor(processor *sentryinbox.Processor) *SentryWebhookWorker {
	owner := uuid.New()
	return &SentryWebhookWorker{
		processor:     processor,
		owner:         owner,
		pollInterval:  sentryWebhookPollInterval,
		batchSize:     sentryWebhookBatchSize,
		concurrency:   sentryWebhookConcurrency,
		leaseDuration: sentryWebhookLeaseDuration,
		leaseRenewal:  sentryWebhookLeaseRenewal,
		logger:        log.WithFields(log.Fields{"worker": "SentryWebhookWorker", "lease_owner": owner.String()}),
	}
}

// WithLease changes the lease timing. Tests use it to expire leases fast.
func (w *SentryWebhookWorker) WithLease(duration, renewal time.Duration) *SentryWebhookWorker {
	w.leaseDuration = duration
	w.leaseRenewal = renewal
	return w
}

func (w *SentryWebhookWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
				w.logger.WithError(err).Error("failed to claim Sentry webhook deliveries")
			}
		}
	}
}

// RunOnce claims one batch and processes it with bounded concurrency. It
// returns the number of claimed entries.
func (w *SentryWebhookWorker) RunOnce(ctx context.Context) (int, error) {
	entries, err := w.claim(ctx)
	if err != nil || len(entries) == 0 {
		return 0, err
	}

	slots := make(chan struct{}, w.concurrency)
	var wg sync.WaitGroup
	for i := range entries {
		entry := entries[i]
		entryCtx, cancel := context.WithCancel(ctx)
		go w.renewLease(entryCtx, cancel, entry)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()

			select {
			case slots <- struct{}{}:
			case <-entryCtx.Done():
				return
			}
			defer func() { <-slots }()
			w.process(entryCtx, &entry)
		}()
	}
	wg.Wait()
	return len(entries), nil
}

func (w *SentryWebhookWorker) claim(ctx context.Context) ([]models.SentryWebhookInbox, error) {
	var entries []models.SentryWebhookInbox
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		claimed, err := models.ClaimSentryWebhookInbox(tx, w.owner, time.Now(), w.leaseDuration, w.batchSize)
		entries = claimed
		return err
	})
	return entries, err
}

func (w *SentryWebhookWorker) process(ctx context.Context, entry *models.SentryWebhookInbox) {
	started := time.Now()
	result, err := w.processor.Process(ctx, entry)
	if err != nil {
		telemetry.RecordSentryWebhookAttempt(context.Background(), time.Since(started), "error")
		w.logger.WithError(err).WithField("inbox_id", entry.ID.String()).
			Warn("Sentry webhook attempt did not record a result; it runs again after the lease expires")
		return
	}
	telemetry.RecordSentryWebhookAttempt(context.Background(), time.Since(started), result.State)
}

// renewLease extends the lease of one entry until its work ends. It works on
// a copy, so it never shares state with the processing goroutine. When the
// lease is lost, it cancels the work and every Sentry request in it.
func (w *SentryWebhookWorker) renewLease(ctx context.Context, cancel context.CancelFunc, entry models.SentryWebhookInbox) {
	ticker := time.NewTicker(w.leaseRenewal)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			held, err := entry.RenewLease(database.DB(ctx), time.Now(), w.leaseDuration)
			if err != nil {
				if ctx.Err() == nil {
					w.logger.WithError(err).WithField("inbox_id", entry.ID.String()).Warn("failed to renew Sentry webhook lease")
				}
				continue
			}
			if !held {
				w.logger.WithField("inbox_id", entry.ID.String()).Warn("Sentry webhook lease was lost; stopping the attempt")
				cancel()
				return
			}
		}
	}
}
