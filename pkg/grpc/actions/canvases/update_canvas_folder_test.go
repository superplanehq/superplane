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

	_, err = UpdateCanvasFolder(ctx, r.Organization.ID.String(), secondFolder.Folder.Metadata.Id, &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{
			Title:           firstFolder.Folder.Spec.Title,
			BackgroundColor: models.CanvasFolderColor1,
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}
