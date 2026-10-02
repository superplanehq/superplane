package middleware

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc"
)

// SetServerError records a handler error for the HTTP Sentry report.
// LoggingMiddleware seeds the holder. A later handler fills it.
func SetServerError(ctx context.Context, err error, tags map[string]string) {
	grpc.SetHTTPHandlerServerError(ctx, err, tags)
}
