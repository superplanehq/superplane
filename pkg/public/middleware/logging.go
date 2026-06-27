package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
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

				fields := log.Fields{
					"method":   r.Method,
					"path":     r.URL.Path,
					"duration": duration,
					"status":   status,
				}

				if ShowFullLogs() {
					fields["remote"] = r.RemoteAddr
					fields["user_agent"] = r.UserAgent()
				}

				logger.WithFields(fields).Info("handled request")

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

func captureHTTPPanic(r *http.Request, status int, recovered any) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetRequest(r)
		scope.SetTag("status", strconv.Itoa(status))
		scope.SetTag("http.path", r.URL.Path)
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

	// Redact UUID-shaped IDs from the URL path so errors from different
	// resources (canvases, executions, integrations, ...) collapse into a
	// single Sentry issue rather than creating a new one per UUID combination.
	groupingPath := groupingPathForRequest(r)

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetRequest(r)
		scope.SetTag("status", strconv.Itoa(status))
		scope.SetTag("http.path", r.URL.Path)
		scope.SetLevel(sentry.LevelWarning)
		hub.CaptureMessage(fmt.Sprintf("HTTP %d %s", status, groupingPath))
	})
}

// uuidPattern matches canonical UUID strings (8-4-4-4-12 hex segments)
// commonly used as path parameters in the SuperPlane API.
var uuidPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)

// groupingPathForRequest returns a path suitable for Sentry issue grouping.
// It redacts UUID-shaped path segments so that the same logical endpoint
// produces a single Sentry issue regardless of the concrete identifiers in
// the URL (e.g. /canvases/{id}/executions/{id}/hooks/approve rather than one
// issue per (canvas, execution) UUID pair).
func groupingPathForRequest(r *http.Request) string {
	return redactPathIDs(r.URL.Path)
}

func redactPathIDs(path string) string {
	return uuidPattern.ReplaceAllString(path, "{id}")
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
