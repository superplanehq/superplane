package grpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/authorization"
)

type denyingPermissionChecker struct{}

func (denyingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

// TestGatewayAuthorizationMiddlewareAuthDeniedDoesNotPanic guards against a
// regression where the middleware captured a still-nil *runtime.ServeMux
// because the variable was passed into runtime.NewServeMux options before
// being assigned. Hitting any authorized route without the required headers
// would then panic inside runtime.MarshalerForRequest on a nil mux, get
// caught by the gateway recovery middleware, and surface in Sentry as a
// "HTTP 500 /api/v1/..." event (alongside the fatal panic capture with
// sentryRecoveryHandler as the culprit). The middleware now reads the mux
// through a getter resolved at request time.
func TestGatewayAuthorizationMiddlewareAuthDeniedDoesNotPanic(t *testing.T) {
	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})

	var mux *runtime.ServeMux
	muxGetter := func() *runtime.ServeMux { return mux }
	mux = runtime.NewServeMux()

	handler := GatewayAuthorizationMiddleware(muxGetter, authorizer)(
		func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
			t.Fatalf("downstream handler should not be invoked on auth failure")
		},
	)

	// /api/v1/service-accounts is in DefaultAuthorizationRules, so it
	// requires auth. With no x-user-id header AuthorizeHTTP fails and the
	// middleware has to render a gateway error response through the mux.
	r := httptest.NewRequest(http.MethodGet, "/api/v1/service-accounts", nil)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		handler(w, r, map[string]string{})
	})

	// The auth-denied path should write a response (HTTPError) rather than
	// crash. We accept any non-2xx response — the key guarantee is that the
	// middleware writes a response without dereferencing a nil mux.
	assert.GreaterOrEqual(t, w.Code, http.StatusBadRequest)
	assert.NotEmpty(t, w.Body.Bytes())
}

// TestGatewayAuthorizationMiddlewareWouldPanicWithNilMux pins down the
// underlying failure mode that caused the production panic: the gateway
// helpers used on the auth-error path crash when the mux is nil. The fix is
// to never let that nil be passed in — the getter pattern accomplishes that.
func TestGatewayAuthorizationMiddlewareWouldPanicWithNilMux(t *testing.T) {
	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/service-accounts", nil)
	w := httptest.NewRecorder()

	route, requiresAuth := authorizer.RouteFromRequest(r)
	assert.True(t, requiresAuth, "/api/v1/service-accounts must require auth for this test to be meaningful")

	_, err := authorizer.AuthorizeHTTP(r.Context(), r, route, map[string]string{})
	assert.Error(t, err, "missing headers should fail authorization")

	assert.Panics(t, func() {
		var nilMux *runtime.ServeMux
		_, outboundMarshaler := runtime.MarshalerForRequest(nilMux, r)
		runtime.HTTPError(r.Context(), nilMux, outboundMarshaler, w, r, err)
	})
}
