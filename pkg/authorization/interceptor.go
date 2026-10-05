package authorization

import "github.com/superplanehq/superplane/pkg/licensing"

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
	LegacyActions                []string
	RequiredExperimentalFeatures []string
	// RequiredLicenseFeatures lists Enterprise features that the installation
	// license must grant. Use it only for operations that create or expand
	// Enterprise access, never for operations that remove access.
	RequiredLicenseFeatures []licensing.Feature
}

func (r AuthorizationRule) AllowedActions() []string {
	return append([]string{r.Action}, r.LegacyActions...)
}
