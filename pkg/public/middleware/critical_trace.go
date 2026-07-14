package middleware

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

type criticalHTTPRouteResolver func(*http.Request) string

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

/*
 * CriticalHTTPTraceMiddleware starts a server span before the handler runs so
 * trace context propagates into grpc-gateway and the internal gRPC server.
 * Only allowlisted routes are traced.
 */
func CriticalHTTPTraceMiddleware(resolveRoute criticalHTTPRouteResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !telemetry.TracingEnabled() || !telemetry.MayTraceHTTPRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			spanName := r.Method + " " + r.URL.Path
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := telemetry.StartSpan(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			capture := &statusCapturingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(capture, r.WithContext(ctx))

			route := resolveRoute(r)
			attrs := []attribute.KeyValue{
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLPathKey.String(r.URL.Path),
				semconv.HTTPResponseStatusCodeKey.Int(capture.statusCode),
			}
			if route != "" {
				attrs = append(attrs, semconv.HTTPRouteKey.String(route))
			}
			if query := r.URL.RawQuery; query != "" {
				attrs = append(attrs, attribute.String("url.query", query))
			}

			span.SetAttributes(attrs...)
			if capture.statusCode >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(capture.statusCode))
			}
		})
	}
}
