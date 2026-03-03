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

func TestResolveCanvasChangeRequestRebasesVersionAndClearsConflicts(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "conflict-resolution-canvas"},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-1",
					Name: "Initial Name",
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

	firstDraftResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	firstDraftID := firstDraftResponse.Version.Metadata.Id

	_, err = UpdateCanvasVersion(
		ctx,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		firstDraftID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "conflict-resolution-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-1",
						Name: "Draft Name",
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
	)
	require.NoError(t, err)

	createRequestResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, firstDraftID)
	require.NoError(t, err)
	changeRequestID := createRequestResponse.ChangeRequest.Metadata.Id

	secondDraftResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	secondDraftID := secondDraftResponse.Version.Metadata.Id

	_, err = UpdateCanvasVersion(
		ctx,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		secondDraftID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "conflict-resolution-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-1",
						Name: "Live Name",
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
	)
	require.NoError(t, err)

	publishSecondDraftResponse, err := PublishCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		secondDraftID,
		"",
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)
	liveVersionID := publishSecondDraftResponse.Version.Metadata.Id

	describeBeforeResolveResponse, err := DescribeCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, changeRequestID)
	require.NoError(t, err)
	assert.Equal(
		t,
		pb.CanvasChangeRequest_STATUS_CONFLICTED,
		describeBeforeResolveResponse.ChangeRequest.Metadata.Status,
	)
	assert.NotEmpty(t, describeBeforeResolveResponse.ChangeRequest.Diff.ConflictingNodeIds)

	resolveResponse, err := ResolveCanvasChangeRequest(
		ctx,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		changeRequestID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "conflict-resolution-canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-1",
						Name: "Resolved Name",
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
	)
	require.NoError(t, err)
	require.NotNil(t, resolveResponse.ChangeRequest)
	require.NotNil(t, resolveResponse.Version)

	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, resolveResponse.ChangeRequest.Metadata.Status)
	assert.Empty(t, resolveResponse.ChangeRequest.Diff.ConflictingNodeIds)
	assert.Equal(t, liveVersionID, resolveResponse.Version.Metadata.BasedOnVersionId)
}
