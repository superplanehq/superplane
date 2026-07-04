package grpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestGatewayAuthorizationMiddlewareReturnsHTTPErrorOnAuthorizationFailure(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(erringPermissionChecker{err: errors.New("db down")})
	middleware := GatewayAuthorizationMiddleware(authorizer)

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
	assert.Contains(t, rec.Body.String(), "internal error")
}

func TestGatewayAuthorizationMiddlewareReturnsNotFoundWhenPermissionDenied(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})
	middleware := GatewayAuthorizationMiddleware(authorizer)

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
	assert.Contains(t, rec.Body.String(), "Not found")
}

type denyingPermissionChecker struct{}

func (denyingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}
