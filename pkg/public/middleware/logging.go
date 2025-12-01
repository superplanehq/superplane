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

				if status >= http.StatusInternalServerError {
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
		hub.Recover(recovered)
		hub.Flush(2 * time.Second)
	})
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
