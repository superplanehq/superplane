package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty name -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			stringPointer("   "),
			stringPointer("description"),
			nil,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("updates canvas metadata", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		newName := support.RandomName("updated-canvas")
		newDescription := "Canvas description updated"

		response, err := UpdateCanvas(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			&newName,
			&newDescription,
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Metadata)
		assert.Equal(t, canvas.ID.String(), response.Canvas.Metadata.Id)
		assert.Equal(t, newName, response.Canvas.Metadata.Name)
		assert.Equal(t, newDescription, response.Canvas.Metadata.Description)

		updatedCanvas, findErr := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, findErr)
		assert.Equal(t, newName, updatedCanvas.Name)
		assert.Equal(t, newDescription, updatedCanvas.Description)
	})

	t.Run("dismisses agent suggestion for the canvas", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		suggestionID := "add-ci"

		response, err := UpdateCanvas(
			context.Background(),
			database.DB(t.Context()),
			canvas,
			nil,
			nil,
			&suggestionID,
		)
		require.NoError(t, err)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Metadata)

		reloaded, err := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"add-ci"}, []string(reloaded.DismissedAgentSuggestionIDs))
		assert.Equal(t, []string{"add-ci"}, response.Canvas.Metadata.DismissedAgentSuggestionIds)

		described, err := DescribeCanvas(context.Background(), database.DB(t.Context()), reloaded)
		require.NoError(t, err)
		assert.Equal(t, []string{"add-ci"}, described.Canvas.Metadata.DismissedAgentSuggestionIds)
	})

	t.Run("duplicate name -> error", func(t *testing.T) {
		existingCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		targetCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			database.DB(t.Context()),
			targetCanvas,
			&existingCanvas.Name,
			&targetCanvas.Description,
			nil,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
	})

}

func stringPointer(value string) *string {
	return &value
}
