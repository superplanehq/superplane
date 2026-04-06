package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListCanvasVersionsPaginated(t *testing.T) {
	r := support.Setup(t)

	t.Run("stale draft versions are cleaned up instead of returning internal errors", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Delete(&models.CanvasVersion{}, "id = ?", draftVersion.ID).Error)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		resp, err := ListCanvasVersionsPaginated(ctx, r.Organization.ID.String(), canvas.ID.String(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Versions, 1)

		var count int64
		require.NoError(
			t,
			database.Conn().
				Model(&models.CanvasUserDraft{}).
				Where("workflow_id = ? AND user_id = ?", canvas.ID, r.User).
				Count(&count).
				Error,
		)
		assert.Zero(t, count)
	})
}
