package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
)

func TestCreateCanvasVersionCreatesUserDraft(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-version"},
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
			},
			Edges: []*componentpb.Edge{},
		},
	})
	require.NoError(t, err)

	canvasID := uuid.MustParse(createCanvasResponse.Canvas.Metadata.Id)
	canvas, err := models.FindCanvas(r.Organization.ID, canvasID)
	require.NoError(t, err)
	require.NotNil(t, canvas.LiveVersionID)

	createVersionResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID.String())
	require.NoError(t, err)
	require.NotNil(t, createVersionResponse.Version)
	require.NotNil(t, createVersionResponse.Version.Metadata)

	metadata := createVersionResponse.Version.Metadata
	assert.Equal(t, canvasID.String(), metadata.CanvasId)
	assert.Equal(t, int32(2), metadata.Revision)
	assert.False(t, metadata.IsPublished)
	assert.Equal(t, canvas.LiveVersionID.String(), metadata.BasedOnVersionId)
	assert.Equal(t, r.User.String(), metadata.Owner.Id)

	var draft models.CanvasUserDraft
	err = database.Conn().Where("workflow_id = ? AND user_id = ?", canvasID, r.User).First(&draft).Error
	require.NoError(t, err)
	assert.Equal(t, metadata.Id, draft.VersionID.String())
}

func TestUpdateCanvasVersionOnlyUpdatesDraft(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-update-version"},
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
			},
			Edges: []*componentpb.Edge{},
		},
	})
	require.NoError(t, err)

	canvasID := createCanvasResponse.Canvas.Metadata.Id
	createVersionResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	versionID := createVersionResponse.Version.Metadata.Id
	canvasUUID := uuid.MustParse(canvasID)
	updateVersionResponse, err := UpdateCanvasVersion(
		ctx,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "canvas-for-update-version"},
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
		},
	)
	require.NoError(t, err)
	require.NotNil(t, updateVersionResponse.Version)

	var nodeCount int64
	err = database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvasUUID, "node-2").
		Count(&nodeCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), nodeCount)

	versionUUID := uuid.MustParse(versionID)
	version, err := models.FindCanvasVersion(canvasUUID, versionUUID)
	require.NoError(t, err)
	assert.Len(t, version.Nodes, 2)
	assert.False(t, version.IsPublished)
}

func TestPublishCanvasVersionAppliesRuntimeChanges(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-publish-version"},
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
			},
			Edges: []*componentpb.Edge{},
		},
	})
	require.NoError(t, err)

	canvasID := createCanvasResponse.Canvas.Metadata.Id
	createVersionResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	versionID := createVersionResponse.Version.Metadata.Id

	_, err = UpdateCanvasVersion(
		ctx,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "canvas-for-publish-version"},
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
		},
	)
	require.NoError(t, err)

	publishResponse, err := PublishCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		"",
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)
	require.NotNil(t, publishResponse.Canvas)
	require.NotNil(t, publishResponse.Version)
	assert.Equal(t, versionID, publishResponse.Version.Metadata.Id)
	assert.True(t, publishResponse.Version.Metadata.IsPublished)

	canvasUUID := uuid.MustParse(canvasID)
	canvas, err := models.FindCanvas(r.Organization.ID, canvasUUID)
	require.NoError(t, err)
	require.NotNil(t, canvas.LiveVersionID)
	assert.Equal(t, versionID, canvas.LiveVersionID.String())

	var nodeCount int64
	err = database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvasUUID, "node-2").
		Count(&nodeCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), nodeCount)

	var draftCount int64
	err = database.Conn().
		Model(&models.CanvasUserDraft{}).
		Where("workflow_id = ? AND user_id = ?", canvasUUID, r.User).
		Count(&draftCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), draftCount)
}
