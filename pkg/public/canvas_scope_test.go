package public

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/datatypes"
)

func Test__AllowsCanvas(t *testing.T) {
	scopedCanvas := uuid.New()
	otherCanvas := uuid.New()

	apiKeyForCanvas := func(canvasIDs ...string) *models.User {
		return &models.User{
			ID:              uuid.New(),
			OrganizationID:  uuid.New(),
			Type:            models.UserTypeAPIKey,
			APIKeyCanvasIDs: datatypes.NewJSONSlice(canvasIDs),
		}
	}

	t.Run("unrestricted credential is allowed on any canvas", func(t *testing.T) {
		user := apiKeyForCanvas()

		require.True(t, allowsCanvas(context.Background(), user, scopedCanvas, "read"))
		require.True(t, allowsCanvas(context.Background(), user, otherCanvas, "read"))
	})

	t.Run("canvas-restricted API key is allowed on its own canvas", func(t *testing.T) {
		user := apiKeyForCanvas(scopedCanvas.String())

		for _, action := range []string{"read", "update", "update_version", "delete"} {
			require.True(t,
				allowsCanvas(context.Background(), user, scopedCanvas, action),
				"action %q should be allowed on the scoped canvas", action,
			)
		}
	})

	t.Run("canvas-restricted API key is denied on another canvas", func(t *testing.T) {
		user := apiKeyForCanvas(scopedCanvas.String())

		for _, action := range []string{"read", "update", "update_version", "delete"} {
			require.False(t,
				allowsCanvas(context.Background(), user, otherCanvas, action),
				"action %q should be denied on a canvas outside the key's scope", action,
			)
		}
	})

	t.Run("scoped token claims are enforced and take precedence", func(t *testing.T) {
		user := apiKeyForCanvas(otherCanvas.String())
		ctx := context.WithValue(
			context.Background(),
			middleware.ScopedTokenClaimsContextKey,
			&jwt.ScopedTokenClaims{
				Scopes: jwt.ScopesFromPermissions([]jwt.Permission{
					{ResourceType: "canvases", Action: "read", Resources: []string{scopedCanvas.String()}},
				}),
			},
		)

		require.True(t, allowsCanvas(ctx, user, scopedCanvas, "read"))
		require.False(t, allowsCanvas(ctx, user, otherCanvas, "read"))
	})

	t.Run("scoped token without a canvases permission is denied", func(t *testing.T) {
		user := apiKeyForCanvas()
		ctx := context.WithValue(
			context.Background(),
			middleware.ScopedTokenClaimsContextKey,
			&jwt.ScopedTokenClaims{
				Scopes: jwt.ScopesFromPermissions([]jwt.Permission{
					{ResourceType: "secrets", Action: "read"},
				}),
			},
		)

		require.False(t, allowsCanvas(ctx, user, scopedCanvas, "read"))
	})
}
