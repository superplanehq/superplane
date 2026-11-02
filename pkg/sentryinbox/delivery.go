package sentryinbox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

// targetOutcome is the explicit result of running one target. A target is
// processed only when the trigger emitted an event.
type targetOutcome struct {
	state  string
	note   string
	node   *models.CanvasNode
	events []bufferedEvent
}

func (o targetOutcome) finished() bool {
	target := models.SentryWebhookDeliveryTarget{State: o.state}
	return target.Finished()
}

func pending(format string, args ...any) targetOutcome {
	return targetOutcome{state: models.SentryWebhookTargetPending, note: fmt.Sprintf(format, args...)}
}

func blocked(note string) targetOutcome {
	return targetOutcome{state: models.SentryWebhookTargetBlocked, note: note}
}

func removed(note string) targetOutcome {
	return targetOutcome{state: models.SentryWebhookTargetRemoved, note: note}
}

func filtered(note string) targetOutcome {
	return targetOutcome{state: models.SentryWebhookTargetFiltered, note: note}
}

// runTarget runs the trigger of one target outside any transaction. It
// buffers emitted events, so the caller can store them together with the
// target state. Enrichment requests to Sentry use the delivery context.
func (p *Processor) runTarget(ctx context.Context, target *models.SentryWebhookDeliveryTarget, message sentryintegration.WebhookMessage) targetOutcome {
	db := database.DB(ctx)

	integration, err := models.FindUnscopedIntegrationInTransaction(db, target.IntegrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return removed("the connection was deleted")
	}
	if err != nil {
		return pending("load connection: %v", err)
	}

	subscription, err := models.FindIntegrationNodeSubscription(db, target.IntegrationID, target.WorkflowID, target.NodeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return removed("the trigger was removed")
	}
	if err != nil {
		return pending("load trigger subscription: %v", err)
	}

	node, err := models.FindCanvasNode(db, target.WorkflowID, target.NodeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return removed("the trigger was removed")
	}
	if err != nil {
		return pending("load trigger: %v", err)
	}

	if outcome, available := checkCanvasAvailable(db, target); !available {
		return outcome
	}

	trigger, err := p.integrationTrigger(subscription)
	if err != nil {
		return blocked(err.Error())
	}

	events := &bufferedEvents{}
	recorder := &outcomeRecorder{}
	err = trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message:       message,
		Configuration: node.Configuration.Data(),
		NodeMetadata:  contexts.NewNodeMetadataContext(db, node),
		Logger:        logging.WithIntegration(logging.ForNode(*node), *integration),
		HTTP:          p.http(ctx),
		Integration:   contexts.NewIntegrationContext(db, nil, integration, p.encryptor, p.registry, nil),
		Events:        events,
		Outcome:       recorder,
		FindExecutionByKV: func(string, string) (*core.ExecutionContext, error) {
			return nil, nil
		},
	})
	if err != nil {
		return pending("the trigger failed: %v", err)
	}
	if recorder.filter != "" {
		return filtered(recorder.filter)
	}
	if len(events.events) == 0 {
		return pending("the trigger returned no event and no filter result")
	}

	return targetOutcome{
		state:  models.SentryWebhookTargetProcessed,
		note:   strings.Join(recorder.warnings, "; "),
		node:   node,
		events: events.events,
	}
}

// checkCanvasAvailable checks the canvas before the trigger runs. A paused
// intake is a configured filter. A canvas without a live version cannot
// start a run, so the target waits until the canvas is published. When the
// canvas is not available, it returns the target outcome and false.
func checkCanvasAvailable(db *gorm.DB, target *models.SentryWebhookDeliveryTarget) (targetOutcome, bool) {
	paused, err := contexts.SkipPausedIntakeFeed(db, target.WorkflowID)
	if err != nil {
		return pending("load intake: %v", err), false
	}
	if paused {
		return filtered("the intake is paused"), false
	}

	_, err = models.FindLiveCanvasVersionInTransaction(db, target.WorkflowID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return blocked("the canvas has no live version"), false
	}
	if err != nil {
		return pending("load canvas version: %v", err), false
	}
	return targetOutcome{}, true
}

func (p *Processor) integrationTrigger(subscription *models.NodeSubscription) (core.IntegrationTrigger, error) {
	if subscription.NodeType != models.NodeTypeTrigger {
		return nil, fmt.Errorf("node type %s cannot receive Sentry webhooks", subscription.NodeType)
	}

	ref := subscription.NodeRef.Data()
	if ref.Trigger == nil {
		return nil, fmt.Errorf("invalid trigger reference")
	}

	trigger, err := p.registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return nil, fmt.Errorf("trigger %s not found", ref.Trigger.Name)
	}

	integrationTrigger, ok := trigger.(core.IntegrationTrigger)
	if !ok {
		return nil, fmt.Errorf("trigger %s cannot receive integration messages", ref.Trigger.Name)
	}
	return integrationTrigger, nil
}

type bufferedEvent struct {
	payloadType string
	payload     any
	dedupKey    string
}

// bufferedEvents keeps emitted events in memory until the completion
// transaction stores them.
type bufferedEvents struct {
	events []bufferedEvent
}

func (b *bufferedEvents) Emit(payloadType string, payload any) error {
	b.events = append(b.events, bufferedEvent{payloadType: payloadType, payload: payload})
	return nil
}

func (b *bufferedEvents) EmitDeduplicated(payloadType string, payload any, dedupKey string) error {
	b.events = append(b.events, bufferedEvent{payloadType: payloadType, payload: payload, dedupKey: dedupKey})
	return nil
}

type outcomeRecorder struct {
	filter   string
	warnings []string
}

func (r *outcomeRecorder) Filter(reason string) {
	r.filter = reason
}

func (r *outcomeRecorder) Warn(message string) {
	r.warnings = append(r.warnings, message)
}
