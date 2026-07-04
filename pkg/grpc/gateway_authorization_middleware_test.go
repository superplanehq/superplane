package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
)

type erringPermissionChecker struct {
	err error
}

func (c erringPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, c.err
}

type denyingPermissionChecker struct{}

func (denyingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

func newTestGatewayMux(t *testing.T) *runtime.ServeMux {
	t.Helper()

	return runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
		runtime.WithErrorHandler(SanitizedGatewayErrorHandler),
	)
}

func TestGatewayAuthorizationMiddlewareReturnsJSONErrorOnAuthorizationFailure(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(erringPermissionChecker{err: errors.New("db down")})

	var mux *runtime.ServeMux
	mux = newTestGatewayMux(t)
	middleware := GatewayAuthorizationMiddleware(&mux, authorizer)

	called := false
	handler := middleware(func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/actions", nil)
	r.Header.Set("x-user-id", "22222222-2222-4222-8222-222222222222")
	r.Header.Set("x-organization-id", "11111111-1111-4111-8111-111111111111")

	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		handler(rec, r, nil)
	})

	assert.False(t, called)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "internal error", body["message"])
}

func TestGatewayAuthorizationMiddlewareReturnsNotFoundWhenPermissionDenied(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})

	var mux *runtime.ServeMux
	mux = newTestGatewayMux(t)
	middleware := GatewayAuthorizationMiddleware(&mux, authorizer)

	called := false
	handler := middleware(func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/actions", nil)
	r.Header.Set("x-user-id", "22222222-2222-4222-8222-222222222222")
	r.Header.Set("x-organization-id", "11111111-1111-4111-8111-111111111111")

	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		handler(rec, r, nil)
	})

	assert.False(t, called)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Not found", body["message"])
}

func TestGatewayAuthorizationMiddlewareResolvesDeferredMuxPointer(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})

	var mux *runtime.ServeMux
	middleware := GatewayAuthorizationMiddleware(&mux, authorizer)
	mux = newTestGatewayMux(t)

	handler := middleware(func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
		t.Fatal("next handler should not be called")
	})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/actions", nil)
	r.Header.Set("x-user-id", "22222222-2222-4222-8222-222222222222")
	r.Header.Set("x-organization-id", "11111111-1111-4111-8111-111111111111")

	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		handler(rec, r, nil)
	})

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}
