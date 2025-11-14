package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func LoggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			// Use a response writer wrapper to capture status code
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(lrw, r)

			duration := time.Since(start)

			if !shouldLogRequest(r.URL.Path) {
				return
			}

			if ShowFullLogs() {
				logger.WithFields(log.Fields{
					"method":     r.Method,
					"path":       r.URL.Path,
					"remote":     r.RemoteAddr,
					"user_agent": r.UserAgent(),
					"duration":   duration,
					"status":     lrw.statusCode,
				}).Info("handled request")
			} else {
				logger.WithFields(log.Fields{
					"method":   r.Method,
					"path":     r.URL.Path,
					"duration": duration,
					"status":   lrw.statusCode,
				}).Info("handled request")
			}
		})
	}
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
