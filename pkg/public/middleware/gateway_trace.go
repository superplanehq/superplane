package middleware

import (
	"context"
	"net/http"

	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

/*
 * TraceGatewayServe wraps grpc-gateway handling so the encode/write tail after
 * the gRPC round trip is visible separately from the otelgrpc client span.
 * Only allowlisted routes are traced, matching CriticalHTTPTraceMiddleware.
 */
func TraceGatewayServe(ctx context.Context, w http.ResponseWriter, gateway http.Handler, r *http.Request) {
	if !telemetry.TracingEnabled() || !telemetry.MayTraceHTTPRequest(r) {
		gateway.ServeHTTP(w, r)
		return
	}

	_ = telemetry.RunSpan(ctx, "grpc_gateway.serve", func(ctx context.Context) error {
		capture := &tracedGatewayResponseWriter{
			ResponseWriter: w,
			ctx:            ctx,
			statusCode:     http.StatusOK,
		}
		gateway.ServeHTTP(capture, r.WithContext(ctx))
		capture.finish()
		return nil
	})
}

type tracedGatewayResponseWriter struct {
	http.ResponseWriter
	ctx         context.Context
	span        trace.Span
	spanStarted bool
	statusCode  int
	bytes       int
}

func (w *tracedGatewayResponseWriter) startSpanIfNeeded() {
	if w.spanStarted || !telemetry.TracingEnabled() {
		return
	}

	w.spanStarted = true
	_, span := telemetry.StartSpan(w.ctx, "grpc_gateway.write_response")
	w.span = span
}

func (w *tracedGatewayResponseWriter) WriteHeader(statusCode int) {
	w.startSpanIfNeeded()
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *tracedGatewayResponseWriter) Write(b []byte) (int, error) {
	w.startSpanIfNeeded()
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *tracedGatewayResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *tracedGatewayResponseWriter) finish() {
	if w.span == nil {
		return
	}

	w.span.SetAttributes(
		semconv.HTTPResponseStatusCodeKey.Int(w.statusCode),
		attribute.Int("http.response.body.size", w.bytes),
	)
	w.span.End()
}
