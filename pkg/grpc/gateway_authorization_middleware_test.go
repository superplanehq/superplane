package grpc

import (
	"context"
	"encoding/json"
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

type denyingPermissionChecker struct{}

func (denyingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

func TestGatewayAuthorizationMiddlewareReturnsJSONErrorOnAuthorizationFailure(t *testing.T) {
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
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "internal error", body["message"])
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
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Not found", body["message"])
}

func TestGatewayAuthorizationMiddlewareDeniesReplayNodeWithoutCanvasesUpdate(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})

	rule, ok := authorizer.Rule(authorization.HTTPRoute{
		Method:  http.MethodPost,
		Pattern: "/api/v1/canvases/{canvas_id}/nodes/{node_id}/replay",
	})
	require.True(t, ok, "ReplayNode route must have an authorization rule registered")
	assert.Equal(t, "canvases", rule.Resource)
	assert.Equal(t, "update", rule.Action)

	middleware := GatewayAuthorizationMiddleware(authorizer)

	handlerCalled := false
	handler := middleware(func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
		handlerCalled = true
	})

	r := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/canvases/11111111-1111-4111-8111-111111111111/nodes/some-node/replay",
		nil,
	)
	r.Header.Set("x-user-id", "22222222-2222-4222-8222-222222222222")
	r.Header.Set("x-organization-id", "11111111-1111-4111-8111-111111111111")

	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		handler(rec, r, map[string]string{"canvas_id": "11111111-1111-4111-8111-111111111111", "node_id": "some-node"})
	})

	assert.False(t, handlerCalled, "handler must not run when canvases:update is denied - it is the only thing that would create a run or queue item")
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Not found", body["message"])
}

func TestGatewayAuthorizationMiddlewareDeniesResolveReplayInputsWithoutCanvasesRead(t *testing.T) {
	t.Parallel()

	authorizer := authorization.NewGatewayAuthorizer(denyingPermissionChecker{})

	rule, ok := authorizer.Rule(authorization.HTTPRoute{
		Method:  http.MethodGet,
		Pattern: "/api/v1/canvases/{canvas_id}/nodes/{node_id}/replay/inputs",
	})
	require.True(t, ok, "ResolveReplayInputs route must have an authorization rule registered")
	assert.Equal(t, "canvases", rule.Resource)
	assert.Equal(t, "read", rule.Action)

	middleware := GatewayAuthorizationMiddleware(authorizer)

	handlerCalled := false
	handler := middleware(func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
		handlerCalled = true
	})

	r := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/canvases/11111111-1111-4111-8111-111111111111/nodes/some-node/replay/inputs",
		nil,
	)
	r.Header.Set("x-user-id", "22222222-2222-4222-8222-222222222222")
	r.Header.Set("x-organization-id", "11111111-1111-4111-8111-111111111111")

	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		handler(rec, r, map[string]string{"canvas_id": "11111111-1111-4111-8111-111111111111", "node_id": "some-node"})
	})

	assert.False(t, handlerCalled, "handler must not run when canvases:read is denied")
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Not found", body["message"])
}
