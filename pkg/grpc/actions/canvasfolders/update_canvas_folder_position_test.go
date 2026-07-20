package canvasfolders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateCanvasFolderPosition__MovesFolderUpAndDown(t *testing.T) {
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

	moveUpResponse, err := UpdateCanvasFolderPosition(
		ctx,
		r.Organization.ID.String(),
		secondFolder.Folder.Metadata.Id,
		pb.UpdateCanvasFolderPositionRequest_DIRECTION_UP,
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

	moveDownResponse, err := UpdateCanvasFolderPosition(
		ctx,
		r.Organization.ID.String(),
		secondFolder.Folder.Metadata.Id,
		pb.UpdateCanvasFolderPositionRequest_DIRECTION_DOWN,
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

func Test__UpdateCanvasFolderPosition__RejectsMissingDirection(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	folder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Production"},
	})
	require.NoError(t, err)

	_, err = UpdateCanvasFolderPosition(
		ctx,
		r.Organization.ID.String(),
		folder.Folder.Metadata.Id,
		pb.UpdateCanvasFolderPositionRequest_DIRECTION_UNSPECIFIED,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}
