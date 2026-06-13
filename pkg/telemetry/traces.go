package telemetry

import (
	"context"
	"os"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	tracesReady    atomic.Bool
)

func InitTracing(ctx context.Context) error {
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return err
	}

	res, err := otelResource(ctx)
	if err != nil {
		return err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	tracerProvider = provider
	tracer = provider.Tracer("superplane")
	tracesReady.Store(true)

	return nil
}

func TracingEnabled() bool {
	return tracesReady.Load()
}

func Tracer() trace.Tracer {
	if tracer != nil {
		return tracer
	}

	return otel.Tracer("superplane")
}

func ShutdownTracing(ctx context.Context) error {
	if tracerProvider == nil {
		return nil
	}

	return tracerProvider.Shutdown(ctx)
}

func serviceName() string {
	name := os.Getenv("OTEL_SERVICE_NAME")
	if name == "" {
		return "superplane"
	}

	return name
}
