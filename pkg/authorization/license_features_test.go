package authorization

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeEntitlements map[licensing.Feature]bool

func (e fakeEntitlements) IsEntitled(feature licensing.Feature) bool {
	return e[feature]
}

func TestLicenseGatedRoutes(t *testing.T) {
	rules := DefaultAuthorizationRules()

	gated := map[HTTPRoute]licensing.Feature{
		{Method: http.MethodPost, Pattern: "/api/v1/roles"}:                     licensing.FeatureCustomRoles,
		{Method: http.MethodPut, Pattern: "/api/v1/roles/{role_name}"}:          licensing.FeatureCustomRoles,
		{Method: http.MethodPost, Pattern: "/api/v1/groups"}:                    licensing.FeatureGroups,
		{Method: http.MethodPut, Pattern: "/api/v1/groups/{group_name}"}:        licensing.FeatureGroups,
		{Method: http.MethodPost, Pattern: "/api/v1/groups/{group_name}/users"}: licensing.FeatureGroups,
	}

	for route, feature := range gated {
		rule, ok := rules[route]
		require.True(t, ok, route.String())
		assert.Equal(t, []licensing.Feature{feature}, rule.RequiredLicenseFeatures, route.String())
	}
}

func TestLicenseDoesNotGateAccessRemovalOrReads(t *testing.T) {
	rules := DefaultAuthorizationRules()

	// Admins must be able to remove Enterprise access after a license expires.
	open := []HTTPRoute{
		{Method: http.MethodDelete, Pattern: "/api/v1/roles/{role_name}"},
		{Method: http.MethodDelete, Pattern: "/api/v1/groups/{group_name}"},
		{Method: http.MethodPatch, Pattern: "/api/v1/groups/{group_name}/users/remove"},
		{Method: http.MethodPost, Pattern: "/api/v1/roles/{role_name}/users"},
		{Method: http.MethodGet, Pattern: "/api/v1/roles/{role_name}"},
		{Method: http.MethodGet, Pattern: "/api/v1/groups/{group_name}/users"},
	}

	for _, route := range open {
		rule, ok := rules[route]
		require.True(t, ok, route.String())
		assert.Empty(t, rule.RequiredLicenseFeatures, route.String())
	}
}

func TestGatewayAuthorizerEnforcesLicenseFeatures(t *testing.T) {
	createRole := HTTPRoute{Method: http.MethodPost, Pattern: "/api/v1/roles"}
	r := httptestRequest(t, map[string]string{
		"x-user-id":         "22222222-2222-4222-8222-222222222222",
		"x-organization-id": "11111111-1111-4111-8111-111111111111",
	})

	t.Run("denies without entitlements", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		authorizer := NewGatewayAuthorizer(allowingPermissionChecker{})

		_, err := authorizer.AuthorizeHTTP(context.Background(), r, createRole, nil)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Equal(t, licensing.ErrNotLicensed.Error(), status.Convert(err).Message())
	})

	t.Run("denies when the license grants a different feature", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		authorizer := NewGatewayAuthorizer(allowingPermissionChecker{}).
			WithEntitlements(fakeEntitlements{licensing.FeatureGroups: true})

		_, err := authorizer.AuthorizeHTTP(context.Background(), r, createRole, nil)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("allows when the license grants the feature", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		authorizer := NewGatewayAuthorizer(allowingPermissionChecker{}).
			WithEntitlements(fakeEntitlements{licensing.FeatureCustomRoles: true})

		_, err := authorizer.AuthorizeHTTP(context.Background(), r, createRole, nil)
		require.NoError(t, err)
	})

	t.Run("nil entitlements keep the Community default", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		authorizer := NewGatewayAuthorizer(allowingPermissionChecker{}).WithEntitlements(nil)

		_, err := authorizer.AuthorizeHTTP(context.Background(), r, createRole, nil)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("RBAC denial hides the license state", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		authorizer := NewGatewayAuthorizer(actionPermissionChecker{})

		_, err := authorizer.AuthorizeHTTP(context.Background(), r, createRole, nil)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}
