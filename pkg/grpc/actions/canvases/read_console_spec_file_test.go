package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__ReadConsoleSpecFile(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("invalid organization id -> error", func(t *testing.T) {
		_, err := ReadRepositorySpecFile(ctx, "not-a-uuid", uuid.New().String(), "", ConsoleYAMLRepositoryPath)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ReadRepositorySpecFile(ctx, orgID, "bad-canvas", "", ConsoleYAMLRepositoryPath)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := ReadRepositorySpecFile(ctx, orgID, uuid.New().String(), "", ConsoleYAMLRepositoryPath)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("empty console when none stored", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		yamlText, err := ReadRepositorySpecFile(ctx, orgID, canvas.ID.String(), "", ConsoleYAMLRepositoryPath)
		require.NoError(t, err)
		doc, err := models.ConsoleFromYML([]byte(yamlText))
		require.NoError(t, err)
		assert.Empty(t, doc.Spec.Panels)
		assert.Empty(t, doc.Spec.Layout)
	})

	t.Run("returns stored panels and layout as console yaml", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		minW, minH := 2, 1

		_, err := models.UpsertCanvasVersionConsole(canvas.ID, []models.ConsolePanel{
			{ID: "p1", Type: "markdown", Content: map[string]any{"body": "hi", "nested": []any{1, map[string]any{"k": "v"}}}},
		}, []models.ConsoleLayoutItem{
			{I: "p1", X: 0, Y: 0, W: 4, H: 2, MinW: &minW, MinH: &minH},
		})

		require.NoError(t, err)

		yamlText, err := ReadRepositorySpecFile(ctx, orgID, canvas.ID.String(), "", ConsoleYAMLRepositoryPath)
		require.NoError(t, err)
		doc, err := models.ConsoleFromYML([]byte(yamlText))
		require.NoError(t, err)
		require.Len(t, doc.Spec.Panels, 1)
		assert.Equal(t, "p1", doc.Spec.Panels[0].ID)
		assert.Equal(t, "markdown", doc.Spec.Panels[0].Type)
		require.NotNil(t, doc.Spec.Panels[0].Content)
		require.Len(t, doc.Spec.Layout, 1)
		assert.Equal(t, "p1", doc.Spec.Layout[0].I)
		assert.EqualValues(t, 4, doc.Spec.Layout[0].W)
		require.NotNil(t, doc.Spec.Layout[0].MinW)
		require.NotNil(t, doc.Spec.Layout[0].MinH)
		assert.EqualValues(t, 2, *doc.Spec.Layout[0].MinW)
		assert.EqualValues(t, 1, *doc.Spec.Layout[0].MinH)
	})
}
