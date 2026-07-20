package middleware

import (
	"context"
	"net/http"

	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
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

	var err error
	ctx, done := telemetry.Span(ctx, "grpc_gateway.serve")
	defer done(&err)

	capture := &tracedGatewayResponseWriter{
		ResponseWriter: w,
		ctx:            ctx,
		statusCode:     http.StatusOK,
	}
	gateway.ServeHTTP(capture, r.WithContext(ctx))
	capture.finish()
}

/*
 * GatewayForwardResponseTraceOption starts a marshal span before grpc-gateway
 * encodes the proto response to JSON. The span ends on the first response write.
 */
func GatewayForwardResponseTraceOption() func(context.Context, http.ResponseWriter, proto.Message) error {
	return func(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
		capture, ok := w.(*tracedGatewayResponseWriter)
		if !ok || !telemetry.TracingEnabled() {
			return nil
		}

		_, span := telemetry.StartSpan(ctx, "grpc_gateway.marshal_response")
		capture.marshalSpan = span
		return nil
	}
}

type tracedGatewayResponseWriter struct {
	http.ResponseWriter
	ctx          context.Context
	marshalSpan  trace.Span
	writeSpan    trace.Span
	writeStarted bool
	statusCode   int
	bytes        int
}

func (w *tracedGatewayResponseWriter) WriteHeader(statusCode int) {
	w.finishMarshalSpan(0)
	w.startWriteSpanIfNeeded()
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *tracedGatewayResponseWriter) Write(b []byte) (int, error) {
	w.finishMarshalSpan(len(b))
	w.startWriteSpanIfNeeded()
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *tracedGatewayResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *tracedGatewayResponseWriter) finishMarshalSpan(bodySize int) {
	if w.marshalSpan == nil {
		return
	}

	if bodySize > 0 {
		w.marshalSpan.SetAttributes(attribute.Int("http.response.body.size", bodySize))
	}
	w.marshalSpan.End()
	w.marshalSpan = nil
}

func (w *tracedGatewayResponseWriter) startWriteSpanIfNeeded() {
	if w.writeStarted || !telemetry.TracingEnabled() {
		return
	}

	w.writeStarted = true
	_, span := telemetry.StartSpan(w.ctx, "grpc_gateway.write_response")
	w.writeSpan = span
}

func (w *tracedGatewayResponseWriter) finish() {
	w.finishMarshalSpan(0)

	if w.writeSpan == nil {
		return
	}

	w.writeSpan.SetAttributes(
		semconv.HTTPResponseStatusCodeKey.Int(w.statusCode),
		attribute.Int("http.response.body.size", w.bytes),
	)
	w.writeSpan.End()
}
