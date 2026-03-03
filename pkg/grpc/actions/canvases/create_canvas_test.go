package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateCanvasDuplicateName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Duplicate Canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)

	_, err = CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestCreateCanvasCreatesLiveVersion(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Versioned Canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)

	canvasID := uuid.MustParse(response.Canvas.Metadata.Id)
	canvas, err := models.FindCanvas(r.Organization.ID, canvasID)
	require.NoError(t, err)
	require.NotNil(t, canvas.LiveVersionID)

	version, err := models.FindCanvasVersion(canvas.ID, *canvas.LiveVersionID)
	require.NoError(t, err)
	require.True(t, version.IsPublished)
	require.Equal(t, 1, version.Revision)
	require.NotNil(t, version.OwnerID)
	require.Equal(t, r.User, *version.OwnerID)
}
