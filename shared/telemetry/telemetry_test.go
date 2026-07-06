package telemetry

import (
	"context"
	"testing"
)

func TestInitDisabledWithoutEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_METRICS_EXPORTER", "")

	shutdown, enabled, err := Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected metrics export disabled")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInitDisabledWhenMetricsExporterNone(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")

	shutdown, enabled, err := Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected metrics export disabled when OTEL_METRICS_EXPORTER=none")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFleetAttr(t *testing.T) {
	attr := FleetAttr("e1-tiny-amd64")
	if attr.Key != "fleet_id" || attr.Value.AsString() != "e1-tiny-amd64" {
		t.Fatalf("FleetAttr: got %v", attr)
	}
}
