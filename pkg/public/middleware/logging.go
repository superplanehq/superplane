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
	"github.com/superplanehq/superplane/pkg/requesterror"
)

func LoggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			// Use a response writer wrapper to capture status code
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Handlers sanitize errors before they reach the client. The
			// recorder keeps the cause, so the error report below contains
			// more than the status code and the path.
			ctx, recorder := requesterror.NewContext(r.Context())
			r = r.WithContext(ctx)

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
					captureHTTPError(r, status, recorder.Err())
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

// captureHTTPError forwards a server error to Sentry. It sends the cause when
// the handler recorded one, because the status code alone does not show why
// the request failed. It sends a message when there is no recorded cause, for
// example when a handler writes a 5xx response without an error.
func captureHTTPError(r *http.Request, status int, err error) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetRequest(r)
		scope.SetTag("status", strconv.Itoa(status))

		if err != nil {
			hub.CaptureException(err)
			return
		}

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
