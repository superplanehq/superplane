package middleware

import (
	"context"
	"net/http"
	"strings"
)

type serverErrorContextKey struct{}

type serverErrorCause struct {
	err  error
	tags map[string]string
}

func withServerErrorCause(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), serverErrorContextKey{}, &serverErrorCause{})
	return r.WithContext(ctx)
}

func serverErrorFrom(ctx context.Context) *serverErrorCause {
	if ctx == nil {
		return nil
	}
	cause, _ := ctx.Value(serverErrorContextKey{}).(*serverErrorCause)
	return cause
}

// SetServerError records a handler error for the HTTP Sentry report.
// LoggingMiddleware seeds the holder. A later handler fills it.
func SetServerError(ctx context.Context, err error, tags map[string]string) {
	cause := serverErrorFrom(ctx)
	if cause == nil || err == nil {
		return
	}

	cause.err = err
	cause.tags = serverErrorTags(tags)
}

func serverErrorTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	copied := make(map[string]string, len(tags))
	for key, value := range tags {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		copied[key] = value
	}
	if len(copied) == 0 {
		return nil
	}
	return copied
}
