package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
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

func TestCreateCanvasInheritsOrganizationVersioningWhenEnabled(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	nowEnabled := true
	require.NoError(t, database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("versioning_enabled", nowEnabled).Error)

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Versioning default canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	// New canvases inherit organization versioning.
	require.True(t, response.Canvas.Metadata.VersioningEnabled)

	require.NotEmpty(t, response.Canvas.Metadata.Id)
	createdCanvasUUID, parseErr := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, parseErr)
	createdCanvas, findErr := models.FindCanvas(r.Organization.ID, createdCanvasUUID)
	require.NoError(t, findErr)
	require.True(t, createdCanvas.VersioningEnabled)
}
