package public

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// requireCanvasRead applies every credential restriction before a read or upgrade.
func (s *Server) requireCanvasRead(ctx context.Context, w http.ResponseWriter, canvasID uuid.UUID) bool {
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return false
	}

	allowed, err := s.authService.CheckOrganizationPermission(ctx,
		user.ID.String(), user.OrganizationID.String(), "canvases", "read")
	if err != nil {
		middleware.SetServerError(ctx, err, nil)
		http.Error(w, "Authorization check failed", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return false
	}
	if user.HasAPIKeyCanvasScope() && !slices.Contains(user.APIKeyCanvasIDs, canvasID.String()) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return false
	}

	// Scoped grants cannot expand the subject's persisted API-key restriction.
	if claims, ok := middleware.GetScopedTokenClaimsFromContext(ctx); ok && claims != nil {
		scopes, err := json.Marshal(claims.Scopes)
		if err != nil {
			middleware.SetServerError(ctx, err, nil)
			http.Error(w, "Authorization check failed", http.StatusInternalServerError)
			return false
		}
		allowed := authorization.HasRequiredScopedTokenPermissionForScopes(string(scopes),
			map[string]string{authorization.CanvasIDPathParam: canvasID.String()},
			authorization.AuthorizationRule{
				Resource: "canvases", Action: "read",
				ResourcePathParams: []string{authorization.CanvasIDPathParam},
			})
		if !allowed {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return false
		}
	}
	return true
}
