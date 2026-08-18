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
type AuthorizationRule struct {
	Resource           string
	Action             string
	DomainType         string
	ResourcePathParams []string
	// LegacyActions keeps persisted grants working during permission migrations.
	// Prefer Action for new checks, and scope legacy actions to the smallest route set possible.
	LegacyActions []string
	// AccountAuthenticated allows account-scoped routes that must not require org membership
	// or RBAC (for example accepting an invite before the account joins the organization).
	// When set, AuthorizeHTTP requires x-account-id and skips org permission checks.
	AccountAuthenticated         bool
	RequiredExperimentalFeatures []string
}

func (r AuthorizationRule) AllowedActions() []string {
	return append([]string{r.Action}, r.LegacyActions...)
}
