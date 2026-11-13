package authorization

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeEntitlements map[licensing.Feature]bool

func (e fakeEntitlements) IsEntitled(_ context.Context, _ uuid.UUID, feature licensing.Feature) bool {
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

func TestLicenseRulesBelongToOrganizations(t *testing.T) {
	for route, rule := range DefaultAuthorizationRules() {
		if len(rule.RequiredLicenseFeatures) == 0 {
			continue
		}
		assert.Equal(t, models.DomainTypeOrganization, rule.DomainType, route.String())
	}
}

func TestGatewayAuthorizerUsesTheOrganizationGrant(t *testing.T) {
	createRole := HTTPRoute{Method: http.MethodPost, Pattern: "/api/v1/roles"}
	granted := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	other := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	authorizer := NewGatewayAuthorizer(allowingPermissionChecker{}).
		WithEntitlements(licensingtest.OrganizationEntitlements{
			granted: {licensing.FeatureCustomRoles},
		})

	t.Run("allows the organization that has the grant", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		request := httptestRequest(t, map[string]string{
			"x-user-id":         "22222222-2222-4222-8222-222222222222",
			"x-organization-id": granted.String(),
		})
		_, err := authorizer.AuthorizeHTTP(context.Background(), request, createRole, nil)
		require.NoError(t, err)
	})

	t.Run("denies an organization without the grant", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		request := httptestRequest(t, map[string]string{
			"x-user-id":         "22222222-2222-4222-8222-222222222222",
			"x-organization-id": other.String(),
		})
		_, err := authorizer.AuthorizeHTTP(context.Background(), request, createRole, nil)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Equal(t, licensing.ErrNotLicensed.Error(), status.Convert(err).Message())
	})

	t.Run("denies a request without an organization", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		request := httptestRequest(t, map[string]string{
			"x-user-id": "22222222-2222-4222-8222-222222222222",
		})
		_, err := authorizer.AuthorizeHTTP(context.Background(), request, createRole, nil)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("denies a request with an invalid organization id", func(t *testing.T) {
		clearOrganizationPermissionCacheForTest()
		request := httptestRequest(t, map[string]string{
			"x-user-id":         "22222222-2222-4222-8222-222222222222",
			"x-organization-id": "not-an-organization",
		})
		_, err := authorizer.AuthorizeHTTP(context.Background(), request, createRole, nil)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}
