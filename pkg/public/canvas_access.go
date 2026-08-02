package public

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func (s *Server) authorizeCanvasRead(ctx context.Context, user *models.User, canvasID uuid.UUID) (bool, error) {
	var scopes []string
	if claims, ok := middleware.GetScopedTokenClaimsFromContext(ctx); ok && claims != nil {
		scopes = claims.Scopes
	}

	var apiKeyCanvasIDs []string
	if user.HasAPIKeyCanvasScope() {
		apiKeyCanvasIDs = append(apiKeyCanvasIDs, user.APIKeyCanvasIDs...)
	}

	return authorization.CheckCanvasAccess(
		ctx,
		s.authService,
		user.ID.String(),
		user.OrganizationID.String(),
		canvasID.String(),
		"canvases",
		"read",
		scopes,
		apiKeyCanvasIDs,
	)
}
