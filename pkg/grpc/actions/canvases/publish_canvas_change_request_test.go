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

func TestPublishCanvasChangeRequestMergesNonConflictingChangesWhenLiveAdvances(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "publish-change-request-merge-canvas"},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-1",
					Name: "Node 1",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "node-2",
					Name: "Node 2",
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

	versionOneResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	versionOneID := versionOneResponse.Version.Metadata.Id

	_, err = UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionOneID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "publish-change-request-merge-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-1",
						Name: "Node 1 updated in version 1",
						Type: componentpb.Node_TYPE_COMPONENT,
						Component: &componentpb.Node_ComponentRef{
							Name: "noop",
						},
					},
					{
						Id:   "node-2",
						Name: "Node 2",
						Type: componentpb.Node_TYPE_COMPONENT,
						Component: &componentpb.Node_ComponentRef{
							Name: "noop",
						},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	changeRequestOneResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, versionOneID)
	require.NoError(t, err)

	versionTwoResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	versionTwoID := versionTwoResponse.Version.Metadata.Id

	_, err = UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionTwoID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "publish-change-request-merge-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-1",
						Name: "Node 1",
						Type: componentpb.Node_TYPE_COMPONENT,
						Component: &componentpb.Node_ComponentRef{
							Name: "noop",
						},
					},
					{
						Id:   "node-2",
						Name: "Node 2 updated in version 2",
						Type: componentpb.Node_TYPE_COMPONENT,
						Component: &componentpb.Node_ComponentRef{
							Name: "noop",
						},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	changeRequestTwoResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, versionTwoID)
	require.NoError(t, err)

	_, err = PublishCanvasChangeRequest(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		changeRequestOneResponse.ChangeRequest.Metadata.Id,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	publishSecondResponse, err := PublishCanvasChangeRequest(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		changeRequestTwoResponse.ChangeRequest.Metadata.Id,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)
	require.NotNil(t, publishSecondResponse.Canvas)
	require.NotNil(t, publishSecondResponse.Canvas.Spec)

	assert.Equal(t, "Node 1 updated in version 1", findNodeNameByID(publishSecondResponse.Canvas.Spec.Nodes, "node-1"))
	assert.Equal(t, "Node 2 updated in version 2", findNodeNameByID(publishSecondResponse.Canvas.Spec.Nodes, "node-2"))
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_PUBLISHED, publishSecondResponse.ChangeRequest.Metadata.Status)
}

func findNodeNameByID(nodes []*componentpb.Node, nodeID string) string {
	for _, node := range nodes {
		if node.GetId() == nodeID {
			return node.GetName()
		}
	}

	return ""
}
