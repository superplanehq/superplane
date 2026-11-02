package sentryinbox

import (
	"context"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

// contextHTTP binds every outgoing request to the context of one delivery.
// A lost lease or a worker shutdown then cancels Sentry API requests.
type contextHTTP struct {
	ctx   context.Context
	inner core.HTTPContext
}

func withContext(ctx context.Context, inner core.HTTPContext) core.HTTPContext {
	if inner == nil {
		return nil
	}
	return contextHTTP{ctx: ctx, inner: inner}
}

func (c contextHTTP) Do(request *http.Request) (*http.Response, error) {
	return c.inner.Do(request.WithContext(c.ctx))
}
