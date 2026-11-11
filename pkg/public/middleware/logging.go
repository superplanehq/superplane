package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
)

func LoggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := requestIDFromHeader(r)
			w.Header().Set(requestIDHeader, requestID)
			r = withRequestLogFields(r, requestID)
			r = grpc.WithServerErrorReport(r)
			// Use a response writer wrapper to capture status code
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			defer func() {
				recovered := recover()
				duration := time.Since(start)

				if !shouldLogRequest(r.URL.Path) {
					if recovered != nil {
						panic(recovered)
					}
					return
				}

				status := lrw.statusCode
				if recovered != nil && status < http.StatusInternalServerError {
					status = http.StatusInternalServerError
				}

				fields := handledRequestFields(r, status, duration)
				logHandledRequest(logger, r, fields, status)

				if recovered != nil {
					captureHTTPPanic(r, status, recovered)
					panic(recovered)
				}

				if shouldCaptureHTTPError(status) {
					captureHTTPError(r, status)
				}
			}()

			next.ServeHTTP(lrw, r)
		})
	}
}

func handledRequestFields(r *http.Request, status int, duration time.Duration) log.Fields {
	fields := log.Fields{
		"method":      r.Method,
		"path":        requestLogPath(r),
		"duration_ms": durationMilliseconds(duration),
		"status":      status,
	}

	if ShowFullLogs() {
		fields["remote"] = r.RemoteAddr
		fields["user_agent"] = r.UserAgent()
	}

	if clientIP := ClientIP(r); clientIP != "" {
		fields["client_ip"] = clientIP
	}

	if traceID := cloudTraceID(r); traceID != "" {
		fields["trace_id"] = traceID
	}

	for key, value := range logging.RequestLogFields(r.Context()) {
		if text, ok := value.(string); ok && text == "" {
			continue
		}
		fields[key] = value
	}
	if _, ok := fields["action"]; !ok {
		fields["action"] = r.Method
	}
	if _, ok := fields["object"]; !ok {
		fields["object"] = fields["path"]
	}
	if _, ok := fields["outcome"]; !ok {
		fields["outcome"] = "success"
	}
	if status >= http.StatusBadRequest {
		fields["outcome"] = "failure"
	}
	return fields
}

// Public links and invitations carry credentials in the path, not just the query.
func requestLogPath(r *http.Request) string {
	for _, prefix := range []string{
		"/api/v1/invitations/", "/api/v1/invite-links/",
		"/api/v1/public/badges/", "/api/v1/public/artifacts/",
	} {
		suffix, ok := strings.CutPrefix(r.URL.Path, prefix)
		if !ok {
			continue
		}
		_, tail, hasTail := strings.Cut(suffix, "/")
		if hasTail {
			return prefix + "[redacted]/" + tail
		}
		return prefix + "[redacted]"
	}
	return r.URL.Path
}

func logHandledRequest(logger *log.Logger, r *http.Request, fields log.Fields, status int) {
	entry := logger.WithFields(fields)
	message := handledRequestMessage(fields)
	if status < http.StatusInternalServerError {
		entry.Info(message)
		return
	}

	if noted, ok := grpc.NotedServerErrorFrom(r.Context()); ok {
		entry = entry.WithError(noted.Cause)
	}
	entry.Error(message)
}

func handledRequestMessage(fields log.Fields) string {
	path, _ := fields["path"].(string)
	if path == "" {
		return "handled request"
	}
	return "handled request " + path
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

// ClientIP uses the same trust order as hosted credit billing.
// Proxy-set headers win. For X-Forwarded-For, the rightmost address is the
// one a trusted proxy appended. The leftmost address is caller-supplied.
func ClientIP(r *http.Request) string {
	for _, header := range []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP"} {
		if ip := strings.TrimSpace(r.Header.Get(header)); ip != "" {
			return ip
		}
	}
	if ip := rightMostForwardedIP(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func rightMostForwardedIP(forwarded string) string {
	parts := strings.Split(forwarded, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if ip := strings.TrimSpace(parts[i]); ip != "" {
			return ip
		}
	}
	return ""
}

// cloudTraceID returns the trace id from the GCP load balancer header.
// Format: TRACE_ID/SPAN_ID;o=TRACE_TRUE
func cloudTraceID(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("X-Cloud-Trace-Context"))
	if header == "" {
		return ""
	}

	traceID, _, _ := strings.Cut(header, "/")
	return strings.TrimSpace(traceID)
}

func captureHTTPPanic(r *http.Request, status int, recovered any) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetRequest(r)
		scope.SetTag("status", strconv.Itoa(status))
		hub.Recover(recovered)
		hub.Flush(2 * time.Second)
	})
}

// shouldCaptureHTTPError reports whether a response status should be sent to Sentry.
func shouldCaptureHTTPError(status int) bool {
	return grpc.ReportableHTTPStatus(status)
}

func captureHTTPError(r *http.Request, status int) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}

	if grpc.ServerErrorAlreadyReported(r.Context()) {
		return
	}

	noted, hasCause := grpc.NotedServerErrorFrom(r.Context())
	hub.WithScope(func(scope *sentry.Scope) {
		request := r
		if noted.OmitRequestBody {
			request = requestWithoutBody(r)
		}
		scope.SetRequest(request)
		scope.SetTag("status", strconv.Itoa(status))
		for key, value := range noted.Tags {
			scope.SetTag(key, value)
		}
		if !hasCause {
			hub.CaptureMessage(fmt.Sprintf("HTTP %d %s", status, r.URL.Path))
			return
		}

		setHandlerMessage(scope, noted.Source)
		hub.CaptureException(noted.Cause)
	})
}

func requestWithoutBody(r *http.Request) *http.Request {
	if r == nil {
		return nil
	}

	clone := r.Clone(r.Context())
	clone.Body = http.NoBody
	clone.GetBody = nil
	clone.ContentLength = 0
	clone.Form = nil
	clone.PostForm = nil
	clone.MultipartForm = nil
	return clone
}

func setHandlerMessage(scope *sentry.Scope, source error) {
	_, message, ok := grpcerrors.HandlerStatus(source)
	if !ok || message == "" {
		return
	}

	scope.SetExtra("handler_message", message)
}

func shouldLogRequest(path string) bool {
	appEnv := os.Getenv("APP_ENV")

	if appEnv != "development" && appEnv != "test" {
		return true
	}

	if strings.HasPrefix(path, "/src/") ||
		strings.HasPrefix(path, "/node_modules/") {
		return false
	}

	return true
}

func ShowFullLogs() bool {
	appEnv := os.Getenv("APP_ENV")
	return appEnv != "development" && appEnv != "test"
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Implement http.Hijacker interface to support WebSocket upgrades
func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := lrw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the ResponseWriter doesn't support hijacking")
	}
	return hijacker.Hijack()
}

// Flush implements [http.Flusher] so streaming handlers (e.g. NDJSON live logs) work
// when this wrapper is the outermost [http.ResponseWriter] seen by the handler.
func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
