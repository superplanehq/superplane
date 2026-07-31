package public

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

//
// A credential can be restricted to a subset of the organization's canvases:
// scoped tokens carry their own scopes, and API keys can be created with an
// explicit canvas list. That restriction lives outside RBAC, so an
// organization-wide permission check does not see it.
//
// grpcGatewayHandler publishes the restriction as x-Token-Scopes and
// GatewayAuthorizer.AuthorizeHTTP enforces it for every route served by the
// gRPC gateway. Routes registered directly on the router never pass through
// either, so they enforce it here instead.
//

// tokenCanvasScopes returns the canvas restriction carried by the request's
// credential, or nil when the credential is unrestricted. It mirrors what
// grpcGatewayHandler writes into x-Token-Scopes.
func tokenCanvasScopes(ctx context.Context, user *models.User) []string {
	if claims, ok := middleware.GetScopedTokenClaimsFromContext(ctx); ok {
		return claims.Scopes
	}

	if user.HasAPIKeyCanvasScope() {
		return apiKeyCanvasScopes(user.APIKeyCanvasIDs)
	}

	return nil
}

// allowsCanvas reports whether the request's credential may perform action on
// canvasID. An unrestricted credential is allowed; the caller is still
// responsible for the RBAC permission check.
func allowsCanvas(ctx context.Context, user *models.User, canvasID uuid.UUID, action string) bool {
	scopes := tokenCanvasScopes(ctx, user)
	if len(scopes) == 0 {
		return true
	}

	return authorization.HasScopedTokenPermission(
		scopes,
		map[string]string{authorization.CanvasIDPathParam: canvasID.String()},
		authorization.AuthorizationRule{
			Resource:           "canvases",
			Action:             action,
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{authorization.CanvasIDPathParam},
		},
	)
}
