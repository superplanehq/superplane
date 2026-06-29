package actions

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const accessActionName = "access"

type accessAction struct {
	auth organizationPermissionChecker
}

func newAccessAction(deps Dependencies) accessAction {
	return accessAction{auth: deps.AuthService}
}

type organizationPermissionChecker interface {
	CheckOrganizationPermission(ctx context.Context, userID, orgID, resource, action string) (bool, error)
}

func (a accessAction) Name() string {
	return accessActionName
}

func (a accessAction) Execute(ctx context.Context, session agents.AgentSessionContext, _ Input) (any, error) {
	if strings.TrimSpace(session.CanvasID) == "" {
		return accessResult{}, fmt.Errorf("session canvas id is required")
	}

	permissions := agents.AgentTokenPermissions(session.CanvasID)
	scopes := agents.AgentTokenScopes(session.CanvasID)
	accessible, unavailable, err := a.apiAccess(ctx, session, permissions)
	if err != nil {
		return accessResult{}, err
	}

	return accessResult{
		Action:         accessActionName,
		CanvasID:       session.CanvasID,
		OrganizationID: session.OrganizationID,
		UserID:         session.UserID,
		TokenScopes:    scopes,
		ToolActions:    a.toolActions(ctx, session, permissions),
		Accessible:     accessible,
		Unavailable:    unavailable,
		Notes: []string{
			"Accessible API routes are the intersection of organization RBAC and the scoped agent token permissions enforced by gateway authorization.",
			"Canvas-scoped token permissions only allow API routes whose authorization rule can resolve this session canvas_id.",
			"Draft graph and Console edits are allowed through canvases:update_version on this canvas; publishing and live app operations are not included in the agent token.",
		},
	}, nil
}

func (a accessAction) apiAccess(ctx context.Context, session agents.AgentSessionContext, permissions []jwt.Permission) ([]apiAccessResult, []apiAccessResult, error) {
	rules := authorization.DefaultAuthorizationRules()
	routes := sortedAuthorizationRoutes(rules)
	rbac := newRBACCache(ctx, a.auth, session.UserID, session.OrganizationID)

	accessible := []apiAccessResult{}
	unavailable := []apiAccessResult{}
	for _, route := range routes {
		rule := rules[route]
		tokenAllowed, resources, tokenReason := scopedTokenAllowsRule(rule, permissions, session.CanvasID)
		rbacAllowed, rbacReason, err := rbac.allows(rule.Resource, rule.Action)
		if err != nil {
			return nil, nil, err
		}

		entry := newAPIAccessResult(route, rule, resources)
		if tokenAllowed && rbacAllowed {
			accessible = append(accessible, entry)
			continue
		}

		entry.Reason = accessDeniedReason(tokenAllowed, tokenReason, rbacAllowed, rbacReason)
		unavailable = append(unavailable, entry)
	}

	return accessible, unavailable, nil
}

func (a accessAction) toolActions(ctx context.Context, session agents.AgentSessionContext, permissions []jwt.Permission) []toolAccessResult {
	rbac := newRBACCache(ctx, a.auth, session.UserID, session.OrganizationID)
	actions := []struct {
		name        string
		resource    string
		operation   string
		scoped      bool
		description string
	}{
		{name: accessActionName, description: "No API permission required; reports this session's token and API route access."},
		{name: readActionName, resource: "canvases", operation: "read", scoped: true},
		{name: readRuntimeActionName, resource: "canvases", operation: "read", scoped: true},
		{name: listFilesActionName, resource: "canvases", operation: "read", scoped: true},
		{name: readFileActionName, resource: "canvases", operation: "read", scoped: true},
		{name: listIntegrationsActionName, resource: "integrations", operation: "read"},
		{name: listResourcesActionName, resource: "integrations", operation: "read"},
		{name: createDraftActionName, resource: "canvases", operation: "update_version", scoped: true},
		{name: writeFileActionName, resource: "canvases", operation: "update_version", scoped: true},
		{name: deleteFileActionName, resource: "canvases", operation: "update_version", scoped: true},
		{name: commitFilesActionName, resource: "canvases", operation: "update_version", scoped: true},
		{name: updateDraftActionName, resource: "canvases", operation: "update_version", scoped: true},
	}

	results := make([]toolAccessResult, 0, len(actions))
	for _, action := range actions {
		if action.name == accessActionName {
			results = append(results, toolAccessResult{Action: action.name, Allowed: true, Reason: action.description})
			continue
		}

		allowedByToken := permissionAllows(permissions, action.resource, action.operation, action.scoped, session.CanvasID)
		allowedByRBAC, rbacReason, err := rbac.allows(action.resource, action.operation)
		result := toolAccessResult{
			Action:   action.name,
			Allowed:  allowedByToken && allowedByRBAC && err == nil,
			Requires: []string{fmt.Sprintf("%s:%s", action.resource, action.operation)},
		}
		if err != nil {
			result.Reason = err.Error()
		} else if !allowedByToken {
			result.Reason = "agent token does not grant the required scope"
		} else if !allowedByRBAC {
			result.Reason = rbacReason
		}
		results = append(results, result)
	}

	return results
}

func sortedAuthorizationRoutes(rules map[authorization.HTTPRoute]authorization.AuthorizationRule) []authorization.HTTPRoute {
	routes := make([]authorization.HTTPRoute, 0, len(rules))
	for route := range rules {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].String() < routes[j].String()
	})
	return routes
}

func scopedTokenAllowsRule(rule authorization.AuthorizationRule, permissions []jwt.Permission, canvasID string) (bool, []string, string) {
	for _, permission := range permissions {
		if permission.ResourceType != rule.Resource || permission.Action != rule.Action {
			continue
		}

		if len(permission.Resources) == 0 {
			return true, nil, ""
		}

		if !slices.Contains(permission.Resources, canvasID) {
			continue
		}

		if len(rule.ResourcePathParams) == 0 {
			return false, nil, "agent token is resource-scoped, but this API route is not resource-scoped by authorization rules"
		}

		return true, []string{canvasID}, ""
	}

	return false, nil, "agent token does not grant this resource and operation"
}

func permissionAllows(permissions []jwt.Permission, resource, operation string, scoped bool, canvasID string) bool {
	for _, permission := range permissions {
		if permission.ResourceType != resource || permission.Action != operation {
			continue
		}
		if len(permission.Resources) == 0 {
			return true
		}
		return scoped && slices.Contains(permission.Resources, canvasID)
	}
	return false
}

func newAPIAccessResult(route authorization.HTTPRoute, rule authorization.AuthorizationRule, resources []string) apiAccessResult {
	return apiAccessResult{
		Method:    route.String(),
		Resource:  rule.Resource,
		Operation: rule.Action,
		Resources: resources,
	}
}

func accessDeniedReason(tokenAllowed bool, tokenReason string, rbacAllowed bool, rbacReason string) string {
	reasons := []string{}
	if !tokenAllowed {
		reasons = append(reasons, tokenReason)
	}
	if !rbacAllowed {
		reasons = append(reasons, rbacReason)
	}
	return strings.Join(reasons, "; ")
}

type rbacCache struct {
	ctx            context.Context
	auth           organizationPermissionChecker
	userID         string
	organizationID string
	decisions      map[string]rbacDecision
}

type rbacDecision struct {
	allowed bool
	reason  string
	err     error
}

func newRBACCache(ctx context.Context, auth organizationPermissionChecker, userID, organizationID string) *rbacCache {
	return &rbacCache{
		ctx:            ctx,
		auth:           auth,
		userID:         userID,
		organizationID: organizationID,
		decisions:      map[string]rbacDecision{},
	}
}

func (c *rbacCache) allows(resource, operation string) (bool, string, error) {
	key := resource + ":" + operation
	if decision, ok := c.decisions[key]; ok {
		return decision.allowed, decision.reason, decision.err
	}

	decision := c.check(resource, operation)
	c.decisions[key] = decision
	return decision.allowed, decision.reason, decision.err
}

func (c *rbacCache) check(resource, operation string) rbacDecision {
	if c.auth == nil {
		return rbacDecision{allowed: false, reason: "authorization service is unavailable"}
	}

	allowed, err := c.auth.CheckOrganizationPermission(c.ctx, c.userID, c.organizationID, resource, operation)
	if err != nil {
		return rbacDecision{err: fmt.Errorf("check RBAC permission %s:%s: %w", resource, operation, err)}
	}
	if !allowed {
		return rbacDecision{reason: "user RBAC does not grant this resource and operation"}
	}
	return rbacDecision{allowed: true}
}
