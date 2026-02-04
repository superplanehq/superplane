package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
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
			Nodes: []*componentpb.NodeDefinition{},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)

	_, err = CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}
