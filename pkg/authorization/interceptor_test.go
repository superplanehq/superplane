package authorization

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestResourceIDsFromPathParams(t *testing.T) {
	t.Run("returns canvas id from canvas_id path param", func(t *testing.T) {
		resourceIDs := resourceIDsFromPathParams(
			map[string]string{"canvas_id": "canvas-123"},
			[]string{CanvasIDPathParam},
		)
		require.Equal(t, []string{"canvas-123"}, resourceIDs)
	})

	t.Run("returns id from id path param", func(t *testing.T) {
		resourceIDs := resourceIDsFromPathParams(
			map[string]string{"id": "canvas-123"},
			[]string{IDPathParam},
		)
		require.Equal(t, []string{"canvas-123"}, resourceIDs)
	})

	t.Run("returns nil when path param is missing", func(t *testing.T) {
		resourceIDs := resourceIDsFromPathParams(map[string]string{}, []string{CanvasIDPathParam})
		assert.Nil(t, resourceIDs)
	})
}

func TestCanvasAuthorizationRulesSeparateStagingAndLiveActions(t *testing.T) {
	rules := DefaultAuthorizationRules()

	tests := []struct {
		route  HTTPRoute
		action string
	}{
		{route: HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{canvas_id}/versions"}, action: "read"},
		{route: HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{canvas_id}/staging"}, action: "read"},
		{route: HTTPRoute{Method: http.MethodPut, Pattern: "/api/v1/canvases/{canvas_id}/staging"}, action: "update_version"},
		{route: HTTPRoute{Method: http.MethodDelete, Pattern: "/api/v1/canvases/{canvas_id}/staging"}, action: "update_version"},
		{route: HTTPRoute{Method: http.MethodPost, Pattern: "/api/v1/canvases/{canvas_id}/staging/commit"}, action: "update_version"},
		{route: HTTPRoute{Method: http.MethodPost, Pattern: "/api/v1/canvases/{canvas_id}/repository/commits"}, action: "update_version"},
		{route: HTTPRoute{Method: http.MethodPut, Pattern: "/api/v1/canvases/{id}"}, action: "update"},
		{route: HTTPRoute{Method: http.MethodDelete, Pattern: "/api/v1/canvases/{id}"}, action: "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.route.String(), func(t *testing.T) {
			rule, ok := rules[tt.route]
			require.True(t, ok)
			assert.Equal(t, "canvases", rule.Resource)
			assert.Equal(t, tt.action, rule.Action)
		})
	}
}

func TestHasRequiredScopedTokenPermissionForScopes(t *testing.T) {
	ruleWithIDPathParam := AuthorizationRule{
		Resource:           "canvases",
		Action:             "read",
		DomainType:         models.DomainTypeOrganization,
		ResourcePathParams: []string{IDPathParam},
	}
	ruleWithCanvasPathParam := AuthorizationRule{
		Resource:           "canvases",
		Action:             "read",
		DomainType:         models.DomainTypeOrganization,
		ResourcePathParams: []string{CanvasIDPathParam},
	}

	tests := []struct {
		name        string
		scopes      string
		pathParams  map[string]string
		rule        AuthorizationRule
		expectAllow bool
	}{
		{
			name:        "allows request without scoped token scopes",
			scopes:      "",
			pathParams:  map[string]string{},
			rule:        ruleWithIDPathParam,
			expectAllow: true,
		},
		{
			name:        "rejects malformed scoped token scopes metadata",
			scopes:      "not-json",
			pathParams:  map[string]string{},
			rule:        ruleWithIDPathParam,
			expectAllow: false,
		},
		{
			name:        "allows matching permission without resource scoping",
			scopes:      marshalScopes(t, []string{"canvases:read"}),
			pathParams:  map[string]string{},
			rule:        ruleWithIDPathParam,
			expectAllow: true,
		},
		{
			name:        "allows matching permission with id path param",
			scopes:      marshalScopes(t, []string{"canvases:read:canvas-123"}),
			pathParams:  map[string]string{"id": "canvas-123"},
			rule:        ruleWithIDPathParam,
			expectAllow: true,
		},
		{
			name:        "rejects resource scoped permission when path param is missing",
			scopes:      marshalScopes(t, []string{"canvases:read:canvas-123"}),
			pathParams:  map[string]string{},
			rule:        ruleWithIDPathParam,
			expectAllow: false,
		},
		{
			name:        "allows matching permission with canvas path param",
			scopes:      marshalScopes(t, []string{"canvases:read:canvas-123"}),
			pathParams:  map[string]string{"canvas_id": "canvas-123"},
			rule:        ruleWithCanvasPathParam,
			expectAllow: true,
		},
		{
			name:        "rejects non matching permission with canvas path param",
			scopes:      marshalScopes(t, []string{"canvases:read:canvas-456"}),
			pathParams:  map[string]string{"canvas_id": "canvas-123"},
			rule:        ruleWithCanvasPathParam,
			expectAllow: false,
		},
		{
			name:        "rejects permission with wrong action",
			scopes:      marshalScopes(t, []string{"canvases:update"}),
			pathParams:  map[string]string{"canvas_id": "canvas-123"},
			rule:        ruleWithCanvasPathParam,
			expectAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := hasRequiredScopedTokenPermissionForScopes(tt.scopes, tt.pathParams, tt.rule)
			assert.Equal(t, tt.expectAllow, allowed)
		})
	}
}

func TestPermissionsFromScopedTokenScopes(t *testing.T) {
	t.Run("parses scoped token scopes", func(t *testing.T) {
		permissions, err := permissionsFromScopedTokenScopes(
			marshalScopes(t, []string{"canvases:read:canvas-123"}),
		)

		require.NoError(t, err)
		require.Len(t, permissions, 1)
		assert.Equal(t, "canvases", permissions[0].ResourceType)
		assert.Equal(t, "read", permissions[0].Action)
		assert.Equal(t, []string{"canvas-123"}, permissions[0].Resources)
	})
}

func TestGatewayAuthorizerSetsOrganizationContext(t *testing.T) {
	authorizer := NewGatewayAuthorizer(allowingPermissionChecker{})
	organizationID := "11111111-1111-4111-8111-111111111111"
	r := httptestRequest(t, map[string]string{
		"x-user-id":         "22222222-2222-4222-8222-222222222222",
		"x-organization-id": organizationID,
	})

	ctx, err := authorizer.AuthorizeHTTP(
		context.Background(),
		r,
		HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{id}"},
		map[string]string{"id": "canvas-123"},
	)
	require.NoError(t, err)
	assert.Equal(t, organizationID, ctx.Value(OrganizationContextKey))
	assert.Equal(t, models.DomainTypeOrganization, ctx.Value(DomainTypeContextKey))
	assert.Equal(t, organizationID, ctx.Value(DomainIdContextKey))
	assert.Equal(t, "canvas-123", PathParamsFromContext(ctx)["id"])
}

func TestAgentRoutesRequireManagedAgentsFeature(t *testing.T) {
	rules := DefaultAuthorizationRules()
	routes := []HTTPRoute{
		{Method: http.MethodGet, Pattern: "/api/v1/agents/canvases/{canvas_id}/chat"},
		{Method: http.MethodPost, Pattern: "/api/v1/agents/canvases/{canvas_id}/chat/reset"},
		{Method: http.MethodPost, Pattern: "/api/v1/agents/chats/{chat_id}/messages"},
		{Method: http.MethodGet, Pattern: "/api/v1/agents/chats/{chat_id}/messages"},
	}

	for _, route := range routes {
		rule, ok := rules[route]
		require.True(t, ok, route.String())
		assert.Equal(t, []string{features.FeatureClaudeManagedAgents}, rule.RequiredExperimentalFeatures)
	}
}

func TestDefaultAuthorizationRulesAreKeyedByHTTPRoute(t *testing.T) {
	rules := DefaultAuthorizationRules()

	rule, ok := rules[HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{id}"}]
	require.True(t, ok)
	assert.Equal(t, "canvases", rule.Resource)
	assert.Equal(t, "read", rule.Action)
	assert.Equal(t, []string{IDPathParam}, rule.ResourcePathParams)
}

func httptestRequest(t *testing.T, headers map[string]string) *http.Request {
	t.Helper()

	r, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	for key, value := range headers {
		r.Header.Set(key, value)
	}

	return r
}

func marshalScopes(t *testing.T, scopes []string) string {
	t.Helper()

	payload, err := json.Marshal(scopes)
	require.NoError(t, err)
	return string(payload)
}
