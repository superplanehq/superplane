package canvases

import (
	"context"
	"errors"
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
	"gorm.io/gorm"
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

func TestCreateCanvasVersionCreatesAnotherDraftForSameUser(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-multiple-drafts"},
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

	firstResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID.String())
	require.NoError(t, err)
	require.NotNil(t, firstResponse.Version)
	require.NotNil(t, firstResponse.Version.Metadata)

	secondResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID.String())
	require.NoError(t, err)
	require.NotNil(t, secondResponse.Version)
	require.NotNil(t, secondResponse.Version.Metadata)

	assert.NotEqual(t, firstResponse.Version.Metadata.Id, secondResponse.Version.Metadata.Id)
	assert.Equal(t, int32(2), firstResponse.Version.Metadata.Revision)
	assert.Equal(t, int32(3), secondResponse.Version.Metadata.Revision)
	assert.Equal(t, canvas.LiveVersionID.String(), firstResponse.Version.Metadata.BasedOnVersionId)
	assert.Equal(t, canvas.LiveVersionID.String(), secondResponse.Version.Metadata.BasedOnVersionId)

	var draft models.CanvasUserDraft
	err = database.Conn().Where("workflow_id = ? AND user_id = ?", canvasID, r.User).First(&draft).Error
	require.NoError(t, err)
	assert.Equal(t, secondResponse.Version.Metadata.Id, draft.VersionID.String())

	versionsResponse, err := ListCanvasVersions(ctx, r.Organization.ID.String(), canvasID.String())
	require.NoError(t, err)
	require.Len(t, versionsResponse.Versions, 3)
	assert.Equal(t, int32(3), versionsResponse.Versions[0].Metadata.Revision)
	assert.Equal(t, int32(2), versionsResponse.Versions[1].Metadata.Revision)
	assert.Equal(t, int32(1), versionsResponse.Versions[2].Metadata.Revision)
}

func TestListCanvasVersionsReturnsVersionsSortedByRevision(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-list-versions"},
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
	_, err = CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	response, err := ListCanvasVersions(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	require.Len(t, response.Versions, 2)

	assert.Equal(t, int32(2), response.Versions[0].Metadata.Revision)
	assert.Equal(t, int32(1), response.Versions[1].Metadata.Revision)
	assert.False(t, response.Versions[0].Metadata.IsPublished)
	assert.True(t, response.Versions[1].Metadata.IsPublished)
}

func TestListCanvasVersionsHidesDraftsFromOtherUsers(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-version-visibility"},
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
	_, err = CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
	_, err = CreateCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	response, err := ListCanvasVersions(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	require.Len(t, response.Versions, 2)

	for _, version := range response.Versions {
		if version.Metadata.IsPublished {
			continue
		}
		assert.Equal(t, r.User.String(), version.Metadata.Owner.Id)
	}
}

func TestListCanvasVersionsShowsOnlyOwnVersionsAndCurrentLive(t *testing.T) {
	r := support.Setup(t)
	userCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(userCtx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-published-version-visibility"},
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
	initialVersionsResponse, err := ListCanvasVersions(userCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	require.Len(t, initialVersionsResponse.Versions, 1)
	initialLiveVersionID := initialVersionsResponse.Versions[0].Metadata.Id

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	firstDraftResponse, err := CreateCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	firstPublishResponse, err := PublishCanvasVersion(
		otherUserCtx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		firstDraftResponse.Version.Metadata.Id,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	secondDraftResponse, err := CreateCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	secondPublishResponse, err := PublishCanvasVersion(
		otherUserCtx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		secondDraftResponse.Version.Metadata.Id,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	firstPublishedVersionID := firstPublishResponse.Version.Metadata.Id
	currentLiveVersionID := secondPublishResponse.Version.Metadata.Id
	require.NotEqual(t, firstPublishedVersionID, currentLiveVersionID)

	response, err := ListCanvasVersions(userCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	require.Len(t, response.Versions, 2)

	versionIDs := []string{response.Versions[0].Metadata.Id, response.Versions[1].Metadata.Id}
	assert.Contains(t, versionIDs, initialLiveVersionID)
	assert.Contains(t, versionIDs, currentLiveVersionID)
	assert.NotContains(t, versionIDs, firstPublishedVersionID)
}

func TestDescribeCanvasVersionReturnsPublishedVersionForAnyUser(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-describe-version"},
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
	versionsResponse, err := ListCanvasVersions(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	require.NotEmpty(t, versionsResponse.Versions)

	var publishedVersionID string
	for _, version := range versionsResponse.Versions {
		if version.Metadata.IsPublished {
			publishedVersionID = version.Metadata.Id
			break
		}
	}
	require.NotEmpty(t, publishedVersionID)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
	describeResponse, err := DescribeCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID, publishedVersionID)
	require.NoError(t, err)
	require.NotNil(t, describeResponse.Version)
	assert.Equal(t, publishedVersionID, describeResponse.Version.Metadata.Id)
	assert.True(t, describeResponse.Version.Metadata.IsPublished)
}

func TestDescribeCanvasVersionBlocksDraftFromOtherUsers(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-describe-draft-visibility"},
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
	draftResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	draftVersionID := draftResponse.Version.Metadata.Id

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
	_, err = DescribeCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version owner mismatch")
}

func TestDescribeCanvasVersionBlocksNonLivePublishedVersionFromOtherUsers(t *testing.T) {
	r := support.Setup(t)
	userCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(userCtx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-describe-published-version-visibility"},
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

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	firstDraftResponse, err := CreateCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	firstPublishResponse, err := PublishCanvasVersion(
		otherUserCtx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		firstDraftResponse.Version.Metadata.Id,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	secondDraftResponse, err := CreateCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	secondPublishResponse, err := PublishCanvasVersion(
		otherUserCtx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		secondDraftResponse.Version.Metadata.Id,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	nonLivePublishedVersionID := firstPublishResponse.Version.Metadata.Id
	livePublishedVersionID := secondPublishResponse.Version.Metadata.Id

	_, err = DescribeCanvasVersion(userCtx, r.Organization.ID.String(), canvasID, nonLivePublishedVersionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version owner mismatch")

	liveResponse, err := DescribeCanvasVersion(userCtx, r.Organization.ID.String(), canvasID, livePublishedVersionID)
	require.NoError(t, err)
	require.NotNil(t, liveResponse.Version)
	assert.Equal(t, livePublishedVersionID, liveResponse.Version.Metadata.Id)
	assert.True(t, liveResponse.Version.Metadata.IsPublished)
}

func TestDiscardCanvasVersionDeletesOwnDraft(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-discard-version"},
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
	draftVersionID := createVersionResponse.Version.Metadata.Id

	_, err = DiscardCanvasVersion(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)

	versionsResponse, err := ListCanvasVersions(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	require.Len(t, versionsResponse.Versions, 1)
	assert.True(t, versionsResponse.Versions[0].Metadata.IsPublished)

	_, err = models.FindCanvasVersion(uuid.MustParse(canvasID), uuid.MustParse(draftVersionID))
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	var draftCount int64
	err = database.Conn().
		Model(&models.CanvasUserDraft{}).
		Where("workflow_id = ? AND user_id = ?", canvasID, r.User).
		Count(&draftCount).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), draftCount)
}

func TestDiscardCanvasVersionRejectsPublished(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-discard-published"},
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
	versionsResponse, err := ListCanvasVersions(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)

	var publishedVersionID string
	for _, version := range versionsResponse.Versions {
		if version.Metadata.IsPublished {
			publishedVersionID = version.Metadata.Id
			break
		}
	}
	require.NotEmpty(t, publishedVersionID)

	_, err = DiscardCanvasVersion(ctx, r.Organization.ID.String(), canvasID, publishedVersionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "published versions are immutable")
}

func TestDiscardCanvasVersionRejectsDraftFromOtherUsers(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-discard-owner-check"},
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
	draftVersionID := createVersionResponse.Version.Metadata.Id

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherUserCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
	_, err = DiscardCanvasVersion(otherUserCtx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version owner mismatch")
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
		nil,
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
		nil,
	)
	require.NoError(t, err)

	publishResponse, err := PublishCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
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

func TestUpdateCanvasVersionAppliesAutoLayout(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: "canvas-for-version-auto-layout"},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-a",
					Name: "Node A",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "node-b",
					Name: "Node B",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{
				{
					SourceId: "node-a",
					TargetId: "node-b",
					Channel:  "default",
				},
			},
		},
	})
	require.NoError(t, err)

	canvasID := createCanvasResponse.Canvas.Metadata.Id
	createVersionResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	versionID := createVersionResponse.Version.Metadata.Id

	updateVersionResponse, err := UpdateCanvasVersion(
		ctx,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "canvas-for-version-auto-layout"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-a",
						Name: "Node A",
						Type: componentpb.Node_TYPE_COMPONENT,
						Position: &componentpb.Position{
							X: 100,
							Y: 100,
						},
						Component: &componentpb.Node_ComponentRef{
							Name: "noop",
						},
					},
					{
						Id:   "node-b",
						Name: "Node B",
						Type: componentpb.Node_TYPE_COMPONENT,
						Position: &componentpb.Position{
							X: 900,
							Y: 900,
						},
						Component: &componentpb.Node_ComponentRef{
							Name: "noop",
						},
					},
				},
				Edges: []*componentpb.Edge{
					{
						SourceId: "node-a",
						TargetId: "node-b",
						Channel:  "default",
					},
				},
			},
		},
		&pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, updateVersionResponse.Version)
	require.NotNil(t, updateVersionResponse.Version.Spec)
	require.Len(t, updateVersionResponse.Version.Spec.Nodes, 2)

	var nodeA *componentpb.Node
	var nodeB *componentpb.Node
	for _, node := range updateVersionResponse.Version.Spec.Nodes {
		if node.GetId() == "node-a" {
			nodeA = node
		}
		if node.GetId() == "node-b" {
			nodeB = node
		}
	}

	require.NotNil(t, nodeA)
	require.NotNil(t, nodeB)
	require.NotNil(t, nodeA.GetPosition())
	require.NotNil(t, nodeB.GetPosition())
	assert.Equal(t, nodeA.GetPosition().GetY(), nodeB.GetPosition().GetY(), "horizontal layout should align nodes by Y")
	assert.Greater(t, nodeB.GetPosition().GetX(), nodeA.GetPosition().GetX(), "downstream node should be placed to the right")
}
