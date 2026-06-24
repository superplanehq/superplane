package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		name := "name"
		description := "description"
		_, err := UpdateCanvas(context.Background(), r.Organization.ID.String(), "invalid-id", &name, &description)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := UpdateCanvas(
			context.Background(),
			r.Organization.ID.String(),
			uuid.New().String(),
			stringPointer("updated-name"),
			stringPointer("updated-description"),
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("empty name -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			r.Organization.ID.String(),
			canvas.ID.String(),
			stringPointer("   "),
			stringPointer("description"),
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
			r.Organization.ID.String(),
			canvas.ID.String(),
			&newName,
			&newDescription,
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

		liveVersion, findVersionErr := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
		require.NoError(t, findVersionErr)
		assert.Equal(t, newName, liveVersion.Name)
		assert.Equal(t, newDescription, liveVersion.Description)
	})

	t.Run("duplicate name -> error", func(t *testing.T) {
		existingCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		targetCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			r.Organization.ID.String(),
			targetCanvas.ID.String(),
			&existingCanvas.Name,
			&targetCanvas.Description,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
	})

}

func stringPointer(value string) *string {
	return &value
}
