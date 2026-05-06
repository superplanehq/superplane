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
		createResponse.Folder.Metadata.Id,
		&pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           "Production Ops",
				BackgroundColor: models.CanvasFolderColor3,
			},
		},
		pb.UpdateCanvasFolderRequest_DIRECTION_UNSPECIFIED,
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
		secondFolder.Folder.Metadata.Id,
		&pb.CanvasFolder{
			Spec: &pb.CanvasFolder_Spec{
				Title:           firstFolder.Folder.Spec.Title,
				BackgroundColor: models.CanvasFolderColor1,
			},
		},
		pb.UpdateCanvasFolderRequest_DIRECTION_UNSPECIFIED,
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
		secondFolder.Folder.Metadata.Id,
		nil,
		pb.UpdateCanvasFolderRequest_DIRECTION_UP,
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
		secondFolder.Folder.Metadata.Id,
		nil,
		pb.UpdateCanvasFolderRequest_DIRECTION_DOWN,
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

func Test__UpdateCanvasFolder__RejectsMixedFieldAndPositionUpdates(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	folder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Production"},
	})
	require.NoError(t, err)

	_, err = UpdateCanvasFolder(
		ctx,
		r.Organization.ID.String(),
		folder.Folder.Metadata.Id,
		&pb.CanvasFolder{Spec: &pb.CanvasFolder_Spec{Title: "Production Ops"}},
		pb.UpdateCanvasFolderRequest_DIRECTION_UP,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
