package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

const testWebhookBaseURL = "http://localhost:3000/api/v1"

func TestCreateCanvasChangeRequestCreatesOpenRequestOnly(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "create-cr-open-only")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")

	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)
	require.NotNil(t, createResponse.ChangeRequest)

	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, createResponse.ChangeRequest.Metadata.Status)
	assert.Nil(t, createResponse.ChangeRequest.Metadata.PublishedAt)
}

func TestActOnCanvasChangeRequestRejectAndReopen(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "reject-reopen")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)
	changeRequestID := createResponse.ChangeRequest.Metadata.Id

	rejectResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REJECT,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_REJECTED, rejectResponse.ChangeRequest.Metadata.Status)

	reopenResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REOPEN,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, reopenResponse.ChangeRequest.Metadata.Status)
}

func TestConflictedChangeRequestCannotBeApproved(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "conflict-approve")

	firstDraftID := createDraftVersion(ctx, t, r, canvasID, "Draft One")
	firstChangeRequestResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, firstDraftID)
	require.NoError(t, err)
	firstChangeRequestID := firstChangeRequestResponse.ChangeRequest.Metadata.Id

	secondDraftID := createDraftVersion(ctx, t, r, canvasID, "Draft Two")
	secondChangeRequestResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, secondDraftID)
	require.NoError(t, err)
	secondChangeRequestID := secondChangeRequestResponse.ChangeRequest.Metadata.Id

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, secondChangeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.NoError(t, err)
	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, secondChangeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH)
	require.NoError(t, err)

	firstChangeRequestDetails, err := DescribeCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, firstChangeRequestID)
	require.NoError(t, err)
	require.NotNil(t, firstChangeRequestDetails.ChangeRequest)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, firstChangeRequestDetails.ChangeRequest.Metadata.Status)
	assert.NotEmpty(t, firstChangeRequestDetails.ChangeRequest.Diff.ConflictingNodeIds)

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, firstChangeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcstatus.Code(err))

	rejectResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		firstChangeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REJECT,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_REJECTED, rejectResponse.ChangeRequest.Metadata.Status)
}

func TestApproveDoesNotPublishUntilPublishAction(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "approve-then-publish")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)
	changeRequestID := createResponse.ChangeRequest.Metadata.Id

	approveResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, approveResponse.ChangeRequest.Metadata.Status)
	require.Len(t, approveResponse.ChangeRequest.Approvals, 1)
	assert.Equal(t, pb.CanvasChangeRequestApproval_STATE_APPROVED, approveResponse.ChangeRequest.Approvals[0].State)

	publishResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_PUBLISHED, publishResponse.ChangeRequest.Metadata.Status)
}

func TestPublishRequiresApprovals(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "publish-needs-approval")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)

	_, err = actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		createResponse.ChangeRequest.Metadata.Id,
		pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH,
	)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcstatus.Code(err))
}

func TestUnapproveRequiresReapprovalBeforePublish(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "unapprove-before-publish")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)
	changeRequestID := createResponse.ChangeRequest.Metadata.Id

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, changeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.NoError(t, err)
	unapproveResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_UNAPPROVE,
	)
	require.NoError(t, err)
	require.Len(t, unapproveResponse.ChangeRequest.Approvals, 2)
	assert.Equal(t, pb.CanvasChangeRequestApproval_STATE_UNAPPROVED, unapproveResponse.ChangeRequest.Approvals[1].State)

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, changeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcstatus.Code(err))

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, changeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.NoError(t, err)
	publishResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_PUBLISHED, publishResponse.ChangeRequest.Metadata.Status)
}

func TestRejectInvalidatesActiveApprovalsAndReopenAllowsApprovalsAgain(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(ctx, t, r, "reject-invalidates-approvals")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)
	changeRequestID := createResponse.ChangeRequest.Metadata.Id

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, changeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.NoError(t, err)

	rejectResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REJECT,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_REJECTED, rejectResponse.ChangeRequest.Metadata.Status)
	require.Len(t, rejectResponse.ChangeRequest.Approvals, 2)
	require.NotNil(t, rejectResponse.ChangeRequest.Approvals[0].InvalidatedAt)
	assert.Equal(t, pb.CanvasChangeRequestApproval_STATE_REJECTED, rejectResponse.ChangeRequest.Approvals[1].State)
	assert.Nil(t, rejectResponse.ChangeRequest.Approvals[1].InvalidatedAt)

	reopenResponse, err := actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		changeRequestID,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REOPEN,
	)
	require.NoError(t, err)
	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, reopenResponse.ChangeRequest.Metadata.Status)
	for _, approval := range reopenResponse.ChangeRequest.Approvals {
		require.NotNil(t, approval.InvalidatedAt)
	}

	_, err = actOnCanvasChangeRequestAction(ctx, r, canvasID, changeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.NoError(t, err)
}

func createCanvasWithNoopNode(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasName string) string {
	t.Helper()
	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("versioning_enabled", true).
			Error,
	)

	createCanvasResponse, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{Name: canvasName},
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
	return createCanvasResponse.Canvas.Metadata.Id
}

func createDraftVersion(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasID string, nodeName string) string {
	t.Helper()

	createVersionResponse, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvasID)
	require.NoError(t, err)
	versionID := createVersionResponse.Version.Metadata.Id

	_, err = UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test Canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:   "node-1",
						Name: nodeName,
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
		testWebhookBaseURL,
	)
	require.NoError(t, err)
	return versionID
}

func actOnCanvasChangeRequestAction(
	ctx context.Context,
	r *support.ResourceRegistry,
	canvasID string,
	changeRequestID string,
	action pb.ActOnCanvasChangeRequestRequest_Action,
) (*pb.ActOnCanvasChangeRequestResponse, error) {
	return ActOnCanvasChangeRequest(
		ctx,
		r.AuthService,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		changeRequestID,
		action,
		testWebhookBaseURL,
	)
}
