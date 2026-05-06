package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateCanvasFolder__UpdatesFolder(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	createResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           "Production",
			BackgroundColor: models.CanvasFolderColor2,
		},
	})
	require.NoError(t, err)

	updateResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		updateFolderRequest(createResponse.Folder.Metadata.Id, &pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           "Production Ops",
				BackgroundColor: models.CanvasFolderColor3,
			},
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, updateResponse.Folder)
	require.NotNil(t, updateResponse.Folder.Spec)
	assert.Equal(t, "Production Ops", updateResponse.Folder.Spec.Title)
	assert.Equal(t, models.CanvasFolderColor3, updateResponse.Folder.Spec.BackgroundColor)
}

func Test__UpdateCanvasFolder__RejectsDuplicateTitle(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Deployments"},
	})
	require.NoError(t, err)

	secondFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Operations"},
	})
	require.NoError(t, err)

	_, err = UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		updateFolderRequest(secondFolder.Folder.Metadata.Id, &pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           firstFolder.Folder.Spec.Title,
				BackgroundColor: models.CanvasFolderColor1,
			},
		}),
	)
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func Test__UpdateCanvasFolder__MovesFolderUpAndDown(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "First"},
	})
	require.NoError(t, err)

	secondFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Second"},
	})
	require.NoError(t, err)

	thirdFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Third"},
	})
	require.NoError(t, err)

	moveUpResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		moveFolderRequest(secondFolder.Folder.Metadata.Id, pb.UpdateCanvasFolderRequest_DIRECTION_UP),
	)
	require.NoError(t, err)
	require.Len(t, moveUpResponse.Folders, 3)
	assert.Equal(t, []string{
		secondFolder.Folder.Metadata.Id,
		thirdFolder.Folder.Metadata.Id,
		firstFolder.Folder.Metadata.Id,
	}, []string{
		moveUpResponse.Folders[0].Metadata.Id,
		moveUpResponse.Folders[1].Metadata.Id,
		moveUpResponse.Folders[2].Metadata.Id,
	})

	moveDownResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		moveFolderRequest(secondFolder.Folder.Metadata.Id, pb.UpdateCanvasFolderRequest_DIRECTION_DOWN),
	)
	require.NoError(t, err)
	require.Len(t, moveDownResponse.Folders, 3)
	assert.Equal(t, []string{
		thirdFolder.Folder.Metadata.Id,
		secondFolder.Folder.Metadata.Id,
		firstFolder.Folder.Metadata.Id,
	}, []string{
		moveDownResponse.Folders[0].Metadata.Id,
		moveDownResponse.Folders[1].Metadata.Id,
		moveDownResponse.Folders[2].Metadata.Id,
	})
}

func Test__UpdateCanvasFolder__RejectsMissingOperation(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	_, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		&pb.UpdateCanvasFolderRequest{},
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func Test__UpdateCanvasFolder__CanAssignAndRemoveCanvasMembership(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	folderResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Team A", BackgroundColor: models.CanvasFolderColor1},
	})
	require.NoError(t, err)
	folderID := folderResponse.Folder.Metadata.Id

	assignResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		membershipRequest([]string{canvas.ID.String()}, folderID),
	)
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

	removeResponse, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		membershipRequest([]string{canvas.ID.String()}, ""),
	)
	require.NoError(t, err)
	require.NotNil(t, removeResponse.Canvas)
	require.NotNil(t, removeResponse.Canvas.Metadata)
	assert.Empty(t, removeResponse.Canvas.Metadata.CanvasFolderId)

	persistedCanvas, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasFolderID)
}

func Test__UpdateCanvasFolder__CanAssignMultipleCanvasesToFolder(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	secondCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	folderResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Team A", BackgroundColor: models.CanvasFolderColor1},
	})
	require.NoError(t, err)
	folderID := folderResponse.Folder.Metadata.Id

	response, err := UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		membershipRequest([]string{firstCanvas.ID.String(), secondCanvas.ID.String()}, folderID),
	)
	require.NoError(t, err)
	require.Len(t, response.Canvases, 2)

	persistedFirstCanvas, err := models.FindCanvas(r.Organization.ID, firstCanvas.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedFirstCanvas.CanvasFolderID)
	assert.Equal(t, folderID, persistedFirstCanvas.CanvasFolderID.String())

	persistedSecondCanvas, err := models.FindCanvas(r.Organization.ID, secondCanvas.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedSecondCanvas.CanvasFolderID)
	assert.Equal(t, folderID, persistedSecondCanvas.CanvasFolderID.String())
}

func updateFolderRequest(id string, folder *pb.CanvasFolder) *pb.UpdateCanvasFolderRequest {
	return &pb.UpdateCanvasFolderRequest{
		Id: id,
		Operation: &pb.UpdateCanvasFolderRequest_Update{
			Update: &pb.UpdateCanvasFolderFields{Folder: folder},
		},
	}
}

func moveFolderRequest(id string, direction pb.UpdateCanvasFolderRequest_Direction) *pb.UpdateCanvasFolderRequest {
	return &pb.UpdateCanvasFolderRequest{
		Id: id,
		Operation: &pb.UpdateCanvasFolderRequest_Move{
			Move: &pb.MoveCanvasFolder{Direction: direction},
		},
	}
}

func membershipRequest(canvasIDs []string, folderID string) *pb.UpdateCanvasFolderRequest {
	return &pb.UpdateCanvasFolderRequest{
		Operation: &pb.UpdateCanvasFolderRequest_Membership{
			Membership: &pb.UpdateCanvasFolderMembership{
				CanvasIds: canvasIDs,
				FolderId:  folderID,
			},
		},
	}
}
