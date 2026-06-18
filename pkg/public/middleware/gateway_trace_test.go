package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceGatewayServeWithoutTracing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, criticalCanvasVersionsPath(), nil)
	rec := httptest.NewRecorder()

	var called bool
	gateway := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte("ok"))
	})

	TraceGatewayServe(req.Context(), rec, gateway, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestTraceGatewayServeRecordsGatewaySpans(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := telemetry.ConfigureTestTracerProvider(exporter)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, criticalCanvasVersionsPath(), nil)
	rec := httptest.NewRecorder()

	gateway := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("payload"))
	})

	TraceGatewayServe(req.Context(), rec, gateway, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "payload", rec.Body.String())

	spanNames := make([]string, 0, len(exporter.GetSpans()))
	for _, span := range exporter.GetSpans() {
		spanNames = append(spanNames, span.Name)
	}

	assert.Contains(t, spanNames, "grpc_gateway.serve")
	assert.Contains(t, spanNames, "grpc_gateway.write_response")
}

func TestGatewayForwardResponseTraceOptionRecordsMarshalSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := telemetry.ConfigureTestTracerProvider(exporter)
	defer cleanup()

	ctx, parent := telemetry.StartSpan(context.Background(), "grpc_gateway.serve")
	defer parent.End()

	capture := &tracedGatewayResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		ctx:            ctx,
		statusCode:     http.StatusOK,
	}

	opt := GatewayForwardResponseTraceOption()
	require.NoError(t, opt(ctx, capture, nil))

	_, err := capture.Write([]byte(`{"ok":true}`))
	require.NoError(t, err)
	capture.finish()

	spanNames := make([]string, 0, len(exporter.GetSpans()))
	for _, span := range exporter.GetSpans() {
		spanNames = append(spanNames, span.Name)
	}

	assert.Contains(t, spanNames, "grpc_gateway.marshal_response")
	assert.Contains(t, spanNames, "grpc_gateway.write_response")
}

func TestTraceGatewayServeSkipsNonCriticalRouteWhenTracingEnabled(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := telemetry.ConfigureTestTracerProvider(exporter)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil)
	rec := httptest.NewRecorder()

	var called bool
	gateway := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	TraceGatewayServe(req.Context(), rec, gateway, req)

	assert.True(t, called)
	assert.Empty(t, exporter.GetSpans())
}

func TestTraceGatewayServeWithoutResponseBody(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := telemetry.ConfigureTestTracerProvider(exporter)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, criticalCanvasVersionsPath(), nil)
	rec := httptest.NewRecorder()

	gateway := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	TraceGatewayServe(req.Context(), rec, gateway, req)

	spanNames := make([]string, 0, len(exporter.GetSpans()))
	for _, span := range exporter.GetSpans() {
		spanNames = append(spanNames, span.Name)
	}

	assert.Contains(t, spanNames, "grpc_gateway.serve")
	assert.NotContains(t, spanNames, "grpc_gateway.write_response")
}

func TestTracedGatewayResponseWriterFlush(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := telemetry.ConfigureTestTracerProvider(exporter)
	defer cleanup()

	base := &flushRecorder{ResponseWriter: httptest.NewRecorder()}
	writer := &tracedGatewayResponseWriter{
		ResponseWriter: base,
		ctx:            context.Background(),
		statusCode:     http.StatusOK,
	}

	writer.Flush()

	assert.True(t, base.flushed)
}

type flushRecorder struct {
	http.ResponseWriter
	flushed bool
}

func (f *flushRecorder) Flush() {
	f.flushed = true
}

func criticalCanvasVersionsPath() string {
	return "/api/v1/canvases/" + uuid.NewString() + "/versions"
}
