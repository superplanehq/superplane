package canvasfolders

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DeleteCanvasFolder__DeletesFolder(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	createResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Production"},
	})
	require.NoError(t, err)

	_, err = DeleteCanvasFolder(ctx, r.Organization.ID.String(), createResponse.Folder.Metadata.Id)
	require.NoError(t, err)

	listResponse, err := ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, listResponse.Folders)
}

func Test__DeleteCanvasFolder__FreesCanvases(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	folderResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Temporary"},
	})
	require.NoError(t, err)
	folderID := folderResponse.Folder.Metadata.Id

	_, err = UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		folderID,
		&pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           "Temporary",
				BackgroundColor: models.CanvasFolderColorBlue,
				Canvases:        []*pb.CanvasRef{{Id: canvas.ID.String()}},
			},
		},
		true,
	)
	require.NoError(t, err)

	_, err = DeleteCanvasFolder(ctx, r.Organization.ID.String(), folderID)
	require.NoError(t, err)

	persistedCanvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvas.ID.String()))
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasFolderID)
}
