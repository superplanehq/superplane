package canvasfolders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"github.com/superplanehq/superplane/test/support"
)

func Test__UpdateCanvasFolderMembership__CanBeAssignedAndRemoved(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	folderResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Team A", BackgroundColor: models.CanvasFolderColorBlue},
	})
	require.NoError(t, err)
	folderID := folderResponse.Folder.Metadata.Id

	assignResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		folderID,
		&pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           "Team A",
				BackgroundColor: models.CanvasFolderColorBlue,
				Canvases:        []*pb.CanvasRef{{Id: canvas.ID.String()}, {Id: otherCanvas.ID.String()}},
			},
		},
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, assignResponse.Folder)
	require.NotNil(t, assignResponse.Folder.Spec)
	assert.Len(t, assignResponse.Folder.Spec.Canvases, 2)

	persistedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedCanvas.CanvasFolderID)
	assert.Equal(t, folderID, persistedCanvas.CanvasFolderID.String())

	listResponse, err := canvases.ListCanvases(ctx, r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Canvases, 2)
	for _, listedCanvas := range listResponse.Canvases {
		assert.Equal(t, folderID, listedCanvas.FolderId)
	}

	removeResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		folderID,
		&pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           "Team A",
				BackgroundColor: models.CanvasFolderColorBlue,
				Canvases:        []*pb.CanvasRef{{Id: otherCanvas.ID.String()}},
			},
		},
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, removeResponse.Folder)
	require.NotNil(t, removeResponse.Folder.Spec)
	assert.Len(t, removeResponse.Folder.Spec.Canvases, 1)
	assert.Equal(t, otherCanvas.ID.String(), removeResponse.Folder.Spec.Canvases[0].Id)

	persistedCanvas, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasFolderID)
}
