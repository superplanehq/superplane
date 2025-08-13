package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListCanvases(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("with organization ID -> list canvases from organization", func(t *testing.T) {
		res, err := ListCanvases(ctx, r.Organization.ID.String(), r.AuthService)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Canvases, 1)
		require.NotNil(t, res.Canvases[0].Metadata)
		assert.Equal(t, r.Canvas.ID.String(), res.Canvases[0].Metadata.Id)
		assert.Equal(t, r.Canvas.Name, res.Canvases[0].Metadata.Name)
		assert.Equal(t, r.Canvas.CreatedBy.String(), res.Canvases[0].Metadata.CreatedBy)
		assert.NotNil(t, res.Canvases[0].Metadata.CreatedAt)
	})

	t.Run("Organization with no canvases -> empty list", func(t *testing.T) {
		res, err := ListCanvases(ctx, uuid.New().String(), r.AuthService)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Canvases)
	})
}
