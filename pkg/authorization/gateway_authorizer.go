package authorization

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GatewayAuthorizer struct {
	auth         organizationPermissionChecker
	rules        map[HTTPRoute]AuthorizationRule
	entitlements licensing.Entitlements
}

func NewGatewayAuthorizer(auth organizationPermissionChecker) *GatewayAuthorizer {
	return &GatewayAuthorizer{
		auth:         auth,
		rules:        DefaultAuthorizationRules(),
		entitlements: licensing.Community,
	}
}

// WithEntitlements sets the license entitlements for rules with
// RequiredLicenseFeatures. Without it, those rules are always denied.
func (a *GatewayAuthorizer) WithEntitlements(entitlements licensing.Entitlements) *GatewayAuthorizer {
	if entitlements != nil {
		a.entitlements = entitlements
	}

	return a
}

func (a *GatewayAuthorizer) Rule(route HTTPRoute) (AuthorizationRule, bool) {
	rule, ok := a.rules[route]
	return rule, ok
}

func (a *GatewayAuthorizer) AuthorizeHTTP(
	ctx context.Context,
	r *http.Request,
	route HTTPRoute,
	pathParams map[string]string,
) (context.Context, error) {
	rule, requiresAuth := a.rules[route]
	if !requiresAuth {
		return withAuthorizedContext(ctx, pathParams, ""), nil
	}

	userID := firstHTTPHeader(r, "x-user-id")
	if userID == "" {
		log.Errorf("User not found in request headers")
		return nil, status.Error(codes.NotFound, "Not found")
	}

	organizationID := firstHTTPHeader(r, "x-organization-id")
	if organizationID == "" {
		log.Errorf("Organization not found in request headers")
		return nil, status.Error(codes.NotFound, "Not found")
	}

	allowed, err := checkOrganizationRulePermission(ctx, a.auth, userID, organizationID, rule)
	if err != nil {
		return nil, err
	}

	if !allowed {
		log.Warnf("User %s tried to %s %s in organization %s", userID, rule.Action, rule.Resource, organizationID)
		return nil, status.Error(codes.NotFound, "Not found")
	}

	if !HasRequiredScopedTokenPermissionForScopes(firstHTTPHeader(r, "x-token-scopes"), pathParams, rule) {
		log.Warnf(
			"Scoped token for user %s is missing required permission %s:%s",
			userID,
			rule.Resource,
			rule.Action,
		)
		return nil, status.Error(codes.NotFound, "Not found")
	}

	if err := checkRequiredExperimentalFeatures(ctx, organizationID, rule); err != nil {
		log.Warnf(
			"User %s tried to access %s:%s in organization %s without required experimental feature",
			userID,
			rule.Resource,
			rule.Action,
			organizationID,
		)
		return nil, err
	}

	for _, feature := range rule.RequiredLicenseFeatures {
		if err := licensing.Require(a.entitlements, feature); err != nil {
			log.Warnf(
				"User %s tried to access %s:%s in organization %s without the %s license feature",
				userID,
				rule.Resource,
				rule.Action,
				organizationID,
				feature,
			)
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	if BlocksSuspendedOrganization(route, rule) {
		id, err := uuid.Parse(organizationID)
		if err != nil {
			return nil, status.Error(codes.NotFound, "Not found")
		}
		organization, err := models.FindOrganizationSuspensionStatus(database.DB(ctx), id)
		if err != nil {
			return nil, status.Error(codes.NotFound, "Not found")
		}
		if organization.IsSuspended() {
			return nil, status.Error(codes.PermissionDenied, models.OrganizationSuspendedMessage)
		}
	}
	return withAuthorizedContext(ctx, pathParams, organizationID), nil
}

// Operation policy is separate from permission actions: cancellation and
// completion retain their existing update permission while suspension permits them.
func BlocksSuspendedOrganization(route HTTPRoute, rule AuthorizationRule) bool {
	switch route.Pattern {
	case "/api/v1/canvases/{canvas_id}/runs/{run_id}/cancel",
		"/api/v1/canvases/{canvas_id}/executions/{execution_id}/cancel",
		"/api/v1/canvases/{canvas_id}/executions/{execution_id}/hooks/{hook_name}",
		"/api/v1/factories/{factory_id}/orders/{order_id}/close",
		"/api/v1/factories/{factory_id}/planning-sessions/{session_id}/end",
		"/api/v1/organizations/{id}/runners/{runner_id}",
		"/api/v1/organizations/{id}/runners/{runner_id}:provider-deleted",
		"/api/v1/organizations/{id}/billing-portal-session":
		return false
	case "/api/v1/factories/{factory_id}/codex-connection":
		return route.Method != "GET" && route.Method != "DELETE"
	case "/api/v1/factories/{factory_id}/velocity/sync":
		return true
	}
	return rule.Action != "read" && rule.Resource != "agents"
}

func withAuthorizedContext(ctx context.Context, pathParams map[string]string, organizationID string) context.Context {
	ctx = WithPathParams(ctx, pathParams)
	if organizationID == "" {
		return ctx
	}

	ctx = context.WithValue(ctx, OrganizationContextKey, organizationID)
	ctx = context.WithValue(ctx, DomainTypeContextKey, models.DomainTypeOrganization)
	ctx = context.WithValue(ctx, DomainIdContextKey, organizationID)
	return ctx
}

func checkOrganizationRulePermission(
	ctx context.Context,
	auth organizationPermissionChecker,
	userID string,
	organizationID string,
	rule AuthorizationRule,
) (bool, error) {
	allowed, err := checkOrganizationPermission(ctx, auth, userID, organizationID, rule.Resource, rule.Action)
	if err != nil || allowed {
		return allowed, err
	}

	for _, action := range rule.LegacyActions {
		allowed, err = checkOrganizationPermission(ctx, auth, userID, organizationID, rule.Resource, action)
		if err != nil || allowed {
			return allowed, err
		}
	}

	return false, nil
}

func firstHTTPHeader(r *http.Request, key string) string {
	if r == nil {
		return ""
	}

	return r.Header.Get(key)
}
