package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
)

func TestCloseCanvasChangeRequestClosesAndCreateReopens(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "close-change-request-canvas"},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-1",
					Name: "noop",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	})
	require.NoError(t, err)
	canvasID := createCanvasResponse.Canvas.Metadata.Id

	createVersionResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	versionID := createVersionResponse.Version.Metadata.Id

	createCRResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, versionID)
	require.NoError(t, err)
	changeRequestID := createCRResponse.ChangeRequest.Metadata.Id
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, createCRResponse.ChangeRequest.Metadata.Status)

	closeResponse, err := CloseCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, changeRequestID)
	require.NoError(t, err)
	require.NotNil(t, closeResponse.ChangeRequest)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_CLOSED, closeResponse.ChangeRequest.Metadata.Status)

	describeResponse, err := DescribeCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, changeRequestID)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_CLOSED, describeResponse.ChangeRequest.Metadata.Status)

	reopenResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, versionID)
	require.NoError(t, err)
	require.NotNil(t, reopenResponse.ChangeRequest)
	assert.Equal(t, changeRequestID, reopenResponse.ChangeRequest.Metadata.Id)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, reopenResponse.ChangeRequest.Metadata.Status)
}
