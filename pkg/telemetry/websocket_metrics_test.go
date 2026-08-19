package telemetry

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRecordWebSocketMetrics_WhenReady(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		metricsReady.Store(false)
	})

	meter = provider.Meter("superplane-test")
	if err := registerWebSocketMetrics(); err != nil {
		t.Fatalf("registerWebSocketMetrics: %v", err)
	}
	metricsReady.Store(true)

	ctx := context.Background()
	RecordWebSocketConnectionOpened(ctx, WebSocketKindCanvas)
	RecordWebSocketConnectionClosed(ctx, WebSocketKindCanvas)
	RecordWebSocketConnectionOutcome(ctx, WebSocketKindAgent, WebSocketConnectionOutcomeAuthError)
	RecordWebSocketBroadcast(ctx, WebSocketKindCanvas, 0)
	RecordWebSocketBroadcast(ctx, WebSocketKindAgent, 3)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	names := map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			names[m.Name] = true
		}
	}

	for _, want := range []string{
		"websocket.connections.open",
		"websocket.connections.total",
		"websocket.broadcasts.total",
		"websocket.broadcast.recipients",
	} {
		if !names[want] {
			t.Fatalf("missing metric %q in %#v", want, names)
		}
	}
}

func TestRecordWebSocketMetrics_NoPanicWhenNotReady(t *testing.T) {
	metricsReady.Store(false)
	ctx := context.Background()
	RecordWebSocketConnectionOpened(ctx, WebSocketKindCanvas)
	RecordWebSocketConnectionClosed(ctx, WebSocketKindCanvas)
	RecordWebSocketConnectionOutcome(ctx, WebSocketKindAgent, WebSocketConnectionOutcomeUpgradeError)
	RecordWebSocketBroadcast(ctx, WebSocketKindCanvas, 2)
}
