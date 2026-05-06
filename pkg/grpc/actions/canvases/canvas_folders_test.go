package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CanvasFolders__CreateListUpdateAndDelete(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	createResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           "  Production  ",
			BackgroundColor: models.CanvasFolderColor2,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, createResponse.Folder)
	require.NotNil(t, createResponse.Folder.Metadata)
	require.NotNil(t, createResponse.Folder.Spec)
	assert.Equal(t, "Production", createResponse.Folder.Spec.Title)
	assert.Equal(t, models.CanvasFolderColor2, createResponse.Folder.Spec.BackgroundColor)

	listResponse, err := ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Folders, 1)
	assert.Equal(t, createResponse.Folder.Metadata.Id, listResponse.Folders[0].Metadata.Id)

	updateResponse, err := UpdateCanvasFolder(ctx, r.Organization.ID.String(), createResponse.Folder.Metadata.Id, &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           "Production Ops",
			BackgroundColor: models.CanvasFolderColor3,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updateResponse.Folder)
	require.NotNil(t, updateResponse.Folder.Spec)
	assert.Equal(t, "Production Ops", updateResponse.Folder.Spec.Title)
	assert.Equal(t, models.CanvasFolderColor3, updateResponse.Folder.Spec.BackgroundColor)

	_, err = DeleteCanvasFolder(ctx, r.Organization.ID.String(), createResponse.Folder.Metadata.Id)
	require.NoError(t, err)

	listResponse, err = ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, listResponse.Folders)
}

func Test__CanvasFolders__Validation(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	_, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "   "},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           "Invalid color",
			BackgroundColor: "red-800",
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title: strings.Repeat("a", 129),
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func Test__CanvasFolders__RejectsDuplicateTitles(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	folder := &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Deployments"},
	}

	_, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), folder)
	require.NoError(t, err)

	_, err = CreateCanvasFolder(ctx, r.Organization.ID.String(), folder)
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func Test__CanvasFolders__RejectsDuplicateTitleOnUpdate(t *testing.T) {
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

	_, err = UpdateCanvasFolder(ctx, r.Organization.ID.String(), secondFolder.Folder.Metadata.Id, &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           firstFolder.Folder.Spec.Title,
			BackgroundColor: models.CanvasFolderColor1,
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func Test__CanvasFolders__AreOrganizationScoped(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	otherOrganization := support.CreateOrganization(t, r, r.User)

	_, err := CreateCanvasFolder(ctx, otherOrganization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Other org"},
	})
	require.NoError(t, err)

	listResponse, err := ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, listResponse.Folders)
}

func Test__CanvasFolders__MembershipCanBeAssignedAndRemoved(t *testing.T) {
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

func Test__CanvasFolders__DeletingFolderFreesCanvases(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	folderResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Temporary"},
	})
	require.NoError(t, err)
	folderID := folderResponse.Folder.Metadata.Id

	_, err = UpdateCanvasFolderMembership(ctx, r.Organization.ID.String(), canvas.ID.String(), folderID)
	require.NoError(t, err)

	_, err = DeleteCanvasFolder(ctx, r.Organization.ID.String(), folderID)
	require.NoError(t, err)

	persistedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasFolderID)
}

func Test__CanvasFolders__ListUsesManualOrderWithNewestFirstByDefault(t *testing.T) {
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

	listResponse, err := ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Folders, 2)
	assert.Equal(t, secondFolder.Folder.Metadata.Id, listResponse.Folders[0].Metadata.Id)
	assert.Equal(t, firstFolder.Folder.Metadata.Id, listResponse.Folders[1].Metadata.Id)
}

func Test__CanvasFolders__MoveUpAndDown(t *testing.T) {
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

	moveUpResponse, err := UpdateCanvasFolderPosition(ctx, r.Organization.ID.String(), secondFolder.Folder.Metadata.Id, pb.UpdateCanvasFolderPositionRequest_DIRECTION_UP)
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

	moveDownResponse, err := UpdateCanvasFolderPosition(ctx, r.Organization.ID.String(), secondFolder.Folder.Metadata.Id, pb.UpdateCanvasFolderPositionRequest_DIRECTION_DOWN)
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
