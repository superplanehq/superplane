package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__UpdateCanvasFolderMembership__CanBeAssignedAndRemoved(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	folderResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Team A", BackgroundColor: models.CanvasFolderColor1},
	})
	require.NoError(t, err)
	folderID := folderResponse.Folder.Metadata.Id

	assignResponse, err := UpdateCanvasFolderMembership(ctx, r.Organization.ID.String(), canvas.ID.String(), folderID)
	require.NoError(t, err)
	require.NotNil(t, assignResponse.Canvas)
	require.NotNil(t, assignResponse.Canvas.Metadata)
	assert.Equal(t, folderID, assignResponse.Canvas.Metadata.CanvasFolderId)

	persistedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedCanvas.CanvasFolderID)
	assert.Equal(t, folderID, persistedCanvas.CanvasFolderID.String())

	listResponse, err := ListCanvases(ctx, r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.Len(t, listResponse.Canvases, 1)
	assert.Equal(t, folderID, listResponse.Canvases[0].Metadata.CanvasFolderId)

	removeResponse, err := UpdateCanvasFolderMembership(ctx, r.Organization.ID.String(), canvas.ID.String(), "")
	require.NoError(t, err)
	require.NotNil(t, removeResponse.Canvas)
	require.NotNil(t, removeResponse.Canvas.Metadata)
	assert.Empty(t, removeResponse.Canvas.Metadata.CanvasFolderId)

	persistedCanvas, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasFolderID)
}
