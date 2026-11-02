package sentryinbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

// Result is the state that one attempt recorded for an inbox entry.
type Result struct {
	State string
	Note  string
}

// Retrying reports whether the entry runs again later.
func (r Result) Retrying() bool {
	return r.State == models.SentryWebhookStateRetrying || r.State == models.SentryWebhookStateBlocked
}

// Processor runs one leased inbox entry. External calls run outside
// database transactions. Each commit first confirms the lease, so a worker
// whose lease expired cannot commit.
type Processor struct {
	encryptor crypto.Encryptor
	registry  *registry.Registry
	httpCtx   core.HTTPContext
	now       func() time.Time
	random    func() float64
	publish   func(*models.CanvasEvent) error
}

func NewProcessor(encryptor crypto.Encryptor, registry *registry.Registry) *Processor {
	processor := &Processor{
		encryptor: encryptor,
		registry:  registry,
		now:       time.Now,
		random:    defaultRandom,
		publish:   messages.PublishCanvasEventCreatedMessage,
	}
	if registry != nil {
		processor.httpCtx = registry.HTTPContext()
	}
	return processor
}

// WithHTTP replaces the outgoing HTTP client. Tests use it to answer Sentry
// API requests.
func (p *Processor) WithHTTP(httpCtx core.HTTPContext) *Processor {
	p.httpCtx = httpCtx
	return p
}

// WithEventPublisher replaces the RabbitMQ notification. Tests use it to
// simulate a broker outage.
func (p *Processor) WithEventPublisher(publish func(*models.CanvasEvent) error) *Processor {
	p.publish = publish
	return p
}

// Process runs one attempt. It returns an error only when it could not
// record a result, for example after a lost lease or a database failure.
// The lease then expires and another attempt runs.
func (p *Processor) Process(ctx context.Context, entry *models.SentryWebhookInbox) (Result, error) {
	body, err := encryption.ForDB(database.DB(ctx), p.encryptor).Decrypt(ctx, entry.PayloadCiphertext, []byte(entry.ID.String()))
	if err != nil {
		return p.reschedule(ctx, entry, models.SentryWebhookStateBlocked, fmt.Sprintf("decrypt payload: %v", err))
	}

	message, err := sentryintegration.ParseWebhookMessage(entry.HookResource, body)
	if err != nil {
		return p.reschedule(ctx, entry, models.SentryWebhookStateBlocked, err.Error())
	}

	if isHostedLifecycleEvent(entry) {
		return p.processInstallation(ctx, entry, body, message.Action)
	}

	message.DeliveryID = entry.ID.String()
	if entry.ReceiptID != nil {
		message.ReceiptID = entry.ReceiptID.String()
	}
	return p.processTargets(ctx, entry, message)
}

func (p *Processor) processInstallation(ctx context.Context, entry *models.SentryWebhookInbox, body []byte, action string) (Result, error) {
	app, ok := sentryintegration.HostedAppFromEnv()
	if !ok || !sentryintegration.HostedInstallEnabled() {
		return p.reschedule(ctx, entry, models.SentryWebhookStateBlocked,
			"this worker cannot process public Sentry app installations: configure SUPERPLANE_SENTRY_APP_SLUG, SUPERPLANE_SENTRY_APP_CLIENT_ID, and SUPERPLANE_SENTRY_APP_CLIENT_SECRET")
	}

	outcome, err := p.runInstallationEvent(ctx, entry, body, app, strings.TrimSpace(action))
	if err != nil {
		state := models.SentryWebhookStateRetrying
		if errors.Is(err, sentryintegration.ErrHostedInstallReconnectRequired) || errors.Is(err, sentryintegration.ErrUnsupportedWebhook) {
			state = models.SentryWebhookStateBlocked
		}
		return p.reschedule(ctx, entry, state, err.Error())
	}

	if outcome.Filtered {
		return p.complete(ctx, entry, models.SentryWebhookStateFiltered, outcome.Note)
	}
	return p.complete(ctx, entry, models.SentryWebhookStateProcessed, outcome.Note)
}

func (p *Processor) runInstallationEvent(
	ctx context.Context,
	entry *models.SentryWebhookInbox,
	body []byte,
	app sentryintegration.HostedApp,
	action string,
) (sentryintegration.HostedInstallationOutcome, error) {
	switch action {
	case "created":
		grant, ok := sentryintegration.ParseInstallationCreatedGrant(entry.HookResource, body)
		if !ok {
			return sentryintegration.HostedInstallationOutcome{}, fmt.Errorf("%w: installation.created has no installation", sentryintegration.ErrUnsupportedWebhook)
		}
		return sentryintegration.ProcessHostedInstallationCreated(ctx, p.http(ctx), app, grant, entry.ReceivedAt)

	case "deleted":
		installationUUID, ok := sentryintegration.ParseInstallationDeletedUUID(entry.HookResource, body)
		if !ok {
			return sentryintegration.HostedInstallationOutcome{}, fmt.Errorf("%w: installation.deleted has no installation", sentryintegration.ErrUnsupportedWebhook)
		}
		return sentryintegration.ProcessHostedInstallationDeleted(installationUUID)

	default:
		return sentryintegration.HostedInstallationOutcome{
			Filtered: true,
			Note:     fmt.Sprintf("the installation action %q needs no work", action),
		}, nil
	}
}

func (p *Processor) processTargets(ctx context.Context, entry *models.SentryWebhookInbox, message sentryintegration.WebhookMessage) (Result, error) {
	if !entry.TargetsResolved {
		resolved, err := p.resolveLateTargets(ctx, entry)
		if err != nil {
			return Result{}, err
		}
		if !resolved {
			return p.finishWithoutConnection(ctx, entry)
		}
	}

	targets, err := entry.ListTargets(database.DB(ctx))
	if err != nil {
		return Result{}, err
	}

	for i := range targets {
		if targets[i].Finished() {
			continue
		}
		if err := p.deliverTarget(ctx, entry, &targets[i], message); err != nil {
			return Result{}, err
		}
	}

	if len(targets) == 0 {
		return p.complete(ctx, entry, models.SentryWebhookStateFiltered,
			fmt.Sprintf("no trigger subscribes to %q events", entry.HookResource))
	}

	summary := summarizeTargets(targets)
	if summary.finished {
		return p.complete(ctx, entry, summary.state, summary.note)
	}
	return p.reschedule(ctx, entry, summary.state, summary.note)
}

// finishWithoutConnection keeps a delivery without a connection blocked, so
// a repaired connection still receives it. When the connection is gone on
// purpose, the delivery finishes as filtered instead.
func (p *Processor) finishWithoutConnection(ctx context.Context, entry *models.SentryWebhookInbox) (Result, error) {
	reason, err := removedConnectionReason(database.DB(ctx), entry)
	if err != nil {
		return p.reschedule(ctx, entry, models.SentryWebhookStateRetrying, fmt.Sprintf("find connection: %v", err))
	}
	if reason != "" {
		return p.complete(ctx, entry, models.SentryWebhookStateFiltered, reason)
	}
	return p.reschedule(ctx, entry, models.SentryWebhookStateBlocked, missingConnectionReason(entry))
}

// resolveLateTargets looks for the connection again. A connection created
// or repaired after acceptance then receives the delivery.
func (p *Processor) resolveLateTargets(ctx context.Context, entry *models.SentryWebhookInbox) (bool, error) {
	now := p.now()
	resolved := false
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := entry.LockLease(tx, now); err != nil {
			return err
		}

		resolution, err := resolveTargets(tx, entry, now)
		if err != nil || !resolution.resolved {
			return err
		}

		resolved = true
		return entry.ResolveTargets(tx, resolution.targets, now)
	})
	return resolved, err
}

func (p *Processor) deliverTarget(
	ctx context.Context,
	entry *models.SentryWebhookInbox,
	target *models.SentryWebhookDeliveryTarget,
	message sentryintegration.WebhookMessage,
) error {
	outcome := p.runTarget(ctx, target, message)
	if outcome.state == models.SentryWebhookTargetProcessed {
		return p.commitEvents(ctx, entry, target, outcome)
	}

	now := p.now()
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := entry.LockLease(tx, now); err != nil {
			return err
		}
		if outcome.finished() {
			return target.Finish(tx, outcome.state, outcome.note, now)
		}
		return target.Defer(tx, outcome.state, outcome.note, now)
	})
}

var errCanvasCannotStartRun = errors.New("the canvas cannot start a run")

// commitEvents stores the events of one target and marks the target
// processed in the same transaction. A crash before the commit leaves
// neither, and a crash after it leaves both, so a retry never emits a
// second event for the target. The deduplication key also protects an
// event that another path already stored.
func (p *Processor) commitEvents(
	ctx context.Context,
	entry *models.SentryWebhookInbox,
	target *models.SentryWebhookDeliveryTarget,
	outcome targetOutcome,
) error {
	now := p.now()
	var created []models.CanvasEvent
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := entry.LockLease(tx, now); err != nil {
			return err
		}

		events := contexts.NewEventContext(tx, outcome.node, nil, func(emitted []models.CanvasEvent) {
			created = append(created, emitted...)
		})
		for _, buffered := range outcome.events {
			result, err := events.EmitWithResult(buffered.payloadType, buffered.payload, buffered.dedupKey)
			if err != nil {
				return fmt.Errorf("store event: %w", err)
			}
			if result == contexts.EmitResultCanvasUnavailable {
				return errCanvasCannotStartRun
			}
		}
		return target.Finish(tx, models.SentryWebhookTargetProcessed, outcome.note, now)
	})

	if errors.Is(err, errCanvasCannotStartRun) {
		return p.deferTarget(ctx, entry, target, models.SentryWebhookTargetBlocked, err.Error())
	}
	if errors.Is(err, models.ErrSentryWebhookLeaseLost) || ctx.Err() != nil {
		return err
	}
	if err != nil {
		return p.deferTarget(ctx, entry, target, models.SentryWebhookTargetPending, err.Error())
	}

	p.notifyEvents(entry, created)
	return nil
}

func (p *Processor) deferTarget(ctx context.Context, entry *models.SentryWebhookInbox, target *models.SentryWebhookDeliveryTarget, state, reason string) error {
	now := p.now()
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := entry.LockLease(tx, now); err != nil {
			return err
		}
		return target.Defer(tx, state, reason, now)
	})
}

// notifyEvents wakes the event router. The events are already durable in
// the pending state, so a RabbitMQ outage only delays them until the router
// polls for pending events.
func (p *Processor) notifyEvents(entry *models.SentryWebhookInbox, events []models.CanvasEvent) {
	for i := range events {
		if err := p.publish(&events[i]); err != nil {
			logging.LogSentryWebhookWarn("Sentry webhook event notification failed; the event router will poll it", log.Fields{
				"inbox_id": entry.ID.String(),
				"event_id": events[i].ID.String(),
				"error":    err.Error(),
			})
		}
	}
}

func (p *Processor) complete(ctx context.Context, entry *models.SentryWebhookInbox, state, note string) (Result, error) {
	now := p.now()
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := entry.LockLease(tx, now); err != nil {
			return err
		}
		return entry.Complete(tx, state, note, now)
	})
	if err != nil {
		return Result{}, err
	}

	logging.LogSentryWebhookInfo("Sentry webhook delivery finished", entryFields(entry))
	return Result{State: state, Note: note}, nil
}

func (p *Processor) reschedule(ctx context.Context, entry *models.SentryWebhookInbox, state, reason string) (Result, error) {
	now := p.now()
	nextAttemptAt := now.Add(RetryDelay(entry.Attempts, p.random))
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := entry.LockLease(tx, now); err != nil {
			return err
		}
		return entry.Reschedule(tx, state, reason, nextAttemptAt, now)
	})
	if err != nil {
		return Result{}, err
	}

	fields := entryFields(entry)
	fields["next_attempt_at"] = nextAttemptAt.UTC().Format(time.RFC3339)
	if state == models.SentryWebhookStateBlocked {
		logging.LogSentryWebhookError("Sentry webhook delivery is blocked", fields, errors.New(reason))
	} else {
		logging.LogSentryWebhookWarn("Sentry webhook delivery will retry", fields)
	}
	return Result{State: state, Note: reason}, nil
}

func (p *Processor) http(ctx context.Context) core.HTTPContext {
	return withContext(ctx, p.httpCtx)
}

func entryFields(entry *models.SentryWebhookInbox) log.Fields {
	return log.Fields{
		"inbox_id":          entry.ID.String(),
		"source":            entry.Source,
		"hook_resource":     entry.HookResource,
		"action":            entry.Action,
		"installation_uuid": entry.InstallationUUID,
		"state":             entry.State,
		"attempts":          entry.Attempts,
		"last_error":        entry.LastError,
	}
}

// removedConnectionReason explains a connection that a user or Sentry
// removed. It returns an empty string when the connection may still appear.
func removedConnectionReason(db *gorm.DB, entry *models.SentryWebhookInbox) (string, error) {
	if entry.Source == models.SentryWebhookSourceHostedApp {
		deleted, err := models.SentryAppInstallationDeleted(db, entry.InstallationUUID)
		if err != nil || !deleted {
			return "", err
		}
		return "Sentry deleted this installation", nil
	}

	if entry.IntegrationID == nil {
		return "", nil
	}
	integration, err := models.FindMaybeDeletedIntegrationInTransaction(db, *entry.IntegrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "the connection was deleted", nil
	}
	if err != nil {
		return "", err
	}
	if integration.DeletedAt.Valid {
		return "the connection was deleted", nil
	}
	return "", nil
}

func missingConnectionReason(entry *models.SentryWebhookInbox) string {
	if entry.Source == models.SentryWebhookSourceHostedApp {
		return fmt.Sprintf("no connection uses Sentry installation %q", entry.InstallationUUID)
	}
	return "the connection that received this webhook was not found"
}

type targetSummary struct {
	finished bool
	state    string
	note     string
}

// summarizeTargets turns target states into the entry state. Unfinished
// target notes come first, so the visible error names the work that waits.
func summarizeTargets(targets []models.SentryWebhookDeliveryTarget) targetSummary {
	summary := targetSummary{finished: true}
	processed := false
	blocked := false
	var waiting, done []string

	for _, target := range targets {
		note := ""
		if target.LastError != "" {
			note = fmt.Sprintf("%s/%s: %s", target.WorkflowID, target.NodeID, target.LastError)
		}
		if target.State == models.SentryWebhookTargetProcessed {
			processed = true
		}
		if target.Finished() {
			if note != "" {
				done = append(done, note)
			}
			continue
		}

		summary.finished = false
		if target.State == models.SentryWebhookTargetBlocked {
			blocked = true
		}
		if note != "" {
			waiting = append(waiting, note)
		}
	}

	summary.note = strings.Join(append(waiting, done...), "; ")
	switch {
	case !summary.finished && blocked:
		summary.state = models.SentryWebhookStateBlocked
	case !summary.finished:
		summary.state = models.SentryWebhookStateRetrying
	case processed:
		summary.state = models.SentryWebhookStateProcessed
	default:
		summary.state = models.SentryWebhookStateFiltered
	}
	return summary
}
