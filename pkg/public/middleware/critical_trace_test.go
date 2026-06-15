package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCriticalHTTPTraceMiddlewareExtractsIncomingTraceContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := telemetry.ConfigureTestTracerProvider(exporter)
	defer cleanup()

	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(previousPropagator)

	parentProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(tracetest.NewInMemoryExporter()))
	parentTracer := parentProvider.Tracer("test")
	parentCtx, parentSpan := parentTracer.Start(t.Context(), "browser.fetch")
	parentSpan.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(parentCtx, carrier)

	req := httptest.NewRequest(http.MethodGet, criticalCanvasPath(), nil)
	for key, value := range carrier {
		req.Header.Set(key, value)
	}

	handler := CriticalHTTPTraceMiddleware(func(*http.Request) string {
		return "/api/v1/canvases/{canvas_id}"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, parentSpan.SpanContext().TraceID(), spans[0].SpanContext.TraceID())
	require.NotNil(t, spans[0].Parent)
	assert.Equal(t, parentSpan.SpanContext().SpanID(), spans[0].Parent.SpanID())
}

func criticalCanvasPath() string {
	return "/api/v1/canvases/" + "550e8400-e29b-41d4-a716-446655440000"
}
