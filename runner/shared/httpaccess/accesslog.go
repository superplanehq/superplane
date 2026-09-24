package httpaccess

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ChiAccessLog emits one structured log line per request (compatible with slog JSON handlers).
//
// Prefer registering after middleware.RequestID and middleware.RealIP so request_id / client IP
// are accurate. Typical order: RequestID → RealIP → ChiAccessLog → Recoverer → routes.
func ChiAccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	if log == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			fields := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("dur", time.Since(start)),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.String("remote", r.RemoteAddr),
			}
			if ua := r.UserAgent(); ua != "" {
				fields = append(fields, slog.String("ua", ua))
			}
			if rid := middleware.GetReqID(r.Context()); rid != "" {
				fields = append(fields, slog.String("request_id", rid))
			}

			log.LogAttrs(r.Context(), slog.LevelInfo, "http_access", fields...)
		})
	}
}
