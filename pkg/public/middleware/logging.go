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
)

func LoggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := requestIDFromHeader(r)
			w.Header().Set(requestIDHeader, requestID)
			r = withRequestLogFields(r, requestID)
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
				logHandledRequest(logger, fields, status)

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
		"path":        r.URL.Path,
		"duration_ms": durationMilliseconds(duration),
		"status":      status,
	}

	if ShowFullLogs() {
		fields["remote"] = r.RemoteAddr
		fields["user_agent"] = r.UserAgent()
	}

	if clientIP := clientIPFromForwardedFor(r); clientIP != "" {
		fields["client_ip"] = clientIP
	}

	if traceID := cloudTraceID(r); traceID != "" {
		fields["trace_id"] = traceID
	}

	appendRequestIdentity(fields, requestLogFieldsFrom(r.Context()))
	return fields
}

func appendRequestIdentity(fields log.Fields, logged *requestLogFields) {
	if logged == nil {
		return
	}

	setLogField(fields, "request_id", logged.requestID)
	setLogField(fields, "organization_id", logged.organizationID)
	setLogField(fields, "user_id", logged.userID)
	setLogField(fields, "account_id", logged.accountID)
	setLogField(fields, "impersonator_account_id", logged.impersonatorAccountID)
}

func setLogField(fields log.Fields, key string, value string) {
	if value == "" {
		return
	}
	fields[key] = value
}

func logHandledRequest(logger *log.Logger, fields log.Fields, status int) {
	entry := logger.WithFields(fields)
	if status >= http.StatusInternalServerError {
		entry.Error("handled request")
		return
	}
	entry.Info("handled request")
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func clientIPFromForwardedFor(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return ""
	}

	clientIP, _, _ := strings.Cut(forwarded, ",")
	return strings.TrimSpace(clientIP)
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

// shouldCaptureHTTPError reports whether a response status code represents a
// server-side error worth forwarding to Sentry.
//
// We only capture true server errors (5xx) and deliberately skip status codes
// that indicate the client sent an unsupported request rather than a real
// server bug. In particular:
//
//   - 501 Not Implemented is returned by grpc-gateway when a path exists but
//     the HTTP method has no mapping (e.g. POST /api/v1/triggers/start when
//     only GET /api/v1/triggers/{name} is defined). Those requests are caused
//     by clients hitting the wrong endpoint and should not create Sentry
//     issues.
//   - 505 HTTP Version Not Supported is likewise a client-caused mismatch.
func shouldCaptureHTTPError(status int) bool {
	if status < http.StatusInternalServerError {
		return false
	}

	switch status {
	case http.StatusNotImplemented, http.StatusHTTPVersionNotSupported:
		return false
	}

	return true
}

func captureHTTPError(r *http.Request, status int) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetRequest(r)
		scope.SetTag("status", strconv.Itoa(status))
		hub.CaptureMessage(fmt.Sprintf("HTTP %d %s", status, r.URL.Path))
	})
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
