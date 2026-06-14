package telemetry

import (
	"context"
	"os"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
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
	/*
	 * Propagate trace context across process boundaries. The default global
	 * propagator is a no-op, so without this the grpc-gateway client span and
	 * the internal gRPC server span end up in separate traces and handler/DB
	 * children never appear under the HTTP request in Dash0.
	 *
	 * TraceContext carries W3C traceparent/tracestate (used by otelgrpc over
	 * gRPC metadata). Baggage is included for standard W3C baggage headers.
	 */
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
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
