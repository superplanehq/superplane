package authorization

import "context"

/*
 * OrganizationIDFromContext returns the organization ID set by the gateway
 * authorization middleware. The middleware sets this key only on requests
 * that go through a route covered by DefaultAuthorizationRules, so handlers
 * for unauthenticated or unmapped routes will see an empty string. Using a
 * comma-ok type assertion here prevents a fatal panic when the context key
 * is missing or holds an unexpected type.
 */
func OrganizationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(OrganizationContextKey).(string)
	return value
}

// DomainIDFromContext returns the domain ID set by the gateway authorization
// middleware, or an empty string when the context value is missing or not a
// string. See OrganizationIDFromContext for the rationale.
func DomainIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(DomainIdContextKey).(string)
	return value
}

// DomainTypeFromContext returns the domain type set by the gateway
// authorization middleware, or an empty string when the context value is
// missing or not a string. See OrganizationIDFromContext for the rationale.
func DomainTypeFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(DomainTypeContextKey).(string)
	return value
}
