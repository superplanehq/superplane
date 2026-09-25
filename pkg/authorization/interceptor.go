package authorization

type contextKey string

const OrganizationContextKey contextKey = "organization"
const DomainTypeContextKey contextKey = "domainType"
const DomainIdContextKey contextKey = "domainId"

const CanvasIDPathParam = "canvas_id"
const IDPathParam = "id"

/*
 * Path parameter keys used to resolve the resource ID referenced by a request.
 * This is used when scoped-tokens are used for authentication / authorization.
 */
// PermissionGrant is one resource and action that can authorize a route.
type PermissionGrant struct {
	Resource string
	Action   string
}

type AuthorizationRule struct {
	Resource           string
	Action             string
	DomainType         string
	ResourcePathParams []string
	// LegacyActions keeps persisted grants working during permission migrations.
	// Prefer Action for new checks, and scope legacy actions to the smallest route set possible.
	LegacyActions []string
	// AlsoAllow keeps an existing grant when the primary permission changes.
	AlsoAllow                    []PermissionGrant
	RequiredExperimentalFeatures []string
}

func (r AuthorizationRule) AllowedActions() []string {
	return append([]string{r.Action}, r.LegacyActions...)
}

func (r AuthorizationRule) Grants() []PermissionGrant {
	grants := make([]PermissionGrant, 0, 1+len(r.LegacyActions)+len(r.AlsoAllow))
	grants = append(grants, PermissionGrant{Resource: r.Resource, Action: r.Action})
	for _, action := range r.LegacyActions {
		grants = append(grants, PermissionGrant{Resource: r.Resource, Action: action})
	}
	return append(grants, r.AlsoAllow...)
}
