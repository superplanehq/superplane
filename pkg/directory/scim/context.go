package scim

import "context"

type orgIDKey struct{}

// WithOrganizationID returns a context carrying the SCIM target organization UUID string.
func WithOrganizationID(ctx context.Context, organizationID string) context.Context {
	return context.WithValue(ctx, orgIDKey{}, organizationID)
}

// OrganizationIDFromContext returns the organization id or empty string.
func OrganizationIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(orgIDKey{}).(string)
	return v
}
