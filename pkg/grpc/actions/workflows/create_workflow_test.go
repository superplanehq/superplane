package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateWorkflowDuplicateName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name: "Duplicate Canvas",
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := CreateWorkflow(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)

	_, err = CreateWorkflow(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}
