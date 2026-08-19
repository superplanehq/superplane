package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	WebSocketKindCanvas = "canvas"
	WebSocketKindAgent  = "agent"

	WebSocketConnectionOutcomeOpened       = "opened"
	WebSocketConnectionOutcomeClosed       = "closed"
	WebSocketConnectionOutcomeUpgradeError = "upgrade_error"
	WebSocketConnectionOutcomeAuthError    = "auth_error"

	WebSocketBroadcastOutcomeDelivered     = "delivered"
	WebSocketBroadcastOutcomeNoSubscribers = "no_subscribers"
)

var (
	websocketConnectionsOpen     metric.Int64UpDownCounter
	websocketConnectionsTotal    metric.Int64Counter
	websocketBroadcastsTotal     metric.Int64Counter
	websocketBroadcastRecipients metric.Int64Histogram
)

func registerWebSocketMetrics() error {
	var err error

	websocketConnectionsOpen, err = meter.Int64UpDownCounter(
		"websocket.connections.open",
		metric.WithDescription("Number of open WebSocket connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	websocketConnectionsTotal, err = meter.Int64Counter(
		"websocket.connections.total",
		metric.WithDescription("WebSocket connection lifecycle outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	websocketBroadcastsTotal, err = meter.Int64Counter(
		"websocket.broadcasts.total",
		metric.WithDescription("WebSocket broadcast outcomes"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	websocketBroadcastRecipients, err = meter.Int64Histogram(
		"websocket.broadcast.recipients",
		metric.WithDescription("Number of recipients per WebSocket broadcast"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	return nil
}

func RecordWebSocketConnectionOpened(ctx context.Context, kind string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(attribute.String("kind", kind))
	websocketConnectionsOpen.Add(ctx, 1, attrs)
	websocketConnectionsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("kind", kind),
			attribute.String("outcome", WebSocketConnectionOutcomeOpened),
		),
	)
}

func RecordWebSocketConnectionClosed(ctx context.Context, kind string) {
	if !metricsReady.Load() {
		return
	}

	attrs := metric.WithAttributes(attribute.String("kind", kind))
	websocketConnectionsOpen.Add(ctx, -1, attrs)
	websocketConnectionsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("kind", kind),
			attribute.String("outcome", WebSocketConnectionOutcomeClosed),
		),
	)
}

func RecordWebSocketConnectionOutcome(ctx context.Context, kind, outcome string) {
	if !metricsReady.Load() {
		return
	}

	websocketConnectionsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("kind", kind),
			attribute.String("outcome", outcome),
		),
	)
}

func RecordWebSocketBroadcast(ctx context.Context, kind string, recipients int) {
	if !metricsReady.Load() {
		return
	}

	outcome := WebSocketBroadcastOutcomeDelivered
	if recipients == 0 {
		outcome = WebSocketBroadcastOutcomeNoSubscribers
	}

	websocketBroadcastsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("kind", kind),
			attribute.String("outcome", outcome),
		),
	)
	websocketBroadcastRecipients.Record(
		ctx,
		int64(recipients),
		metric.WithAttributes(attribute.String("kind", kind)),
	)
}
