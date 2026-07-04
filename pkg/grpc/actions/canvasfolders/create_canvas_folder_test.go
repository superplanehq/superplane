package canvasfolders

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__CreateCanvasFolder__CreatesFolder(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	createResponse, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           "  Production  ",
			BackgroundColor: models.CanvasFolderColorGreen,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, createResponse.Folder)
	require.NotNil(t, createResponse.Folder.Metadata)
	require.NotNil(t, createResponse.Folder.Spec)
	assert.Equal(t, "Production", createResponse.Folder.Spec.Title)
	assert.Equal(t, models.CanvasFolderColorGreen, createResponse.Folder.Spec.BackgroundColor)
}

func Test__CreateCanvasFolder__Validation(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	_, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "   "},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

	_, err = CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           "Invalid color",
			BackgroundColor: "red-800",
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

	_, err = CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title: strings.Repeat("a", 129),
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func Test__CreateCanvasFolder__RejectsDuplicateTitles(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	folder := &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Deployments"},
	}

	_, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), folder)
	require.NoError(t, err)

	_, err = CreateCanvasFolder(ctx, r.Organization.ID.String(), folder)
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}
