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
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestCreateCanvasChangeRequestCreatesOpenRequestOnly(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "create-cr-open-only")
	createDraftVersion(ctx, t, r, canvasID, "Draft Name")

	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
	require.NoError(t, err)
	require.NotNil(t, createResponse.ChangeRequest)

	assert.Equal(t, pb.CanvasChangeRequest_STATUS_OPEN, createResponse.ChangeRequest.Metadata.Status)
	assert.Nil(t, createResponse.ChangeRequest.Metadata.PublishedAt)
}

func TestActOnCanvasChangeRequestRejectAndReopen(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "reject-reopen")
	createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
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

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "conflict-approve")

	createDraftVersion(ctx, t, r, canvasID, "Draft One")
	firstChangeRequestResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
	require.NoError(t, err)
	require.NotEmpty(t, firstChangeRequestResponse.ChangeRequest.Diff.ChangedNodeIds)
	firstChangeRequestID := firstChangeRequestResponse.ChangeRequest.Metadata.Id

	secondUser := support.CreateUser(t, r, r.Organization.ID)
	secondUserCtx := authentication.SetUserIdInMetadata(context.Background(), secondUser.ID.String())
	createDraftVersion(secondUserCtx, t, r, canvasID, "Draft Two")
	secondChangeRequestResponse, err := CreateCanvasChangeRequest(secondUserCtx, r.Organization.ID.String(), canvasID, "")
	require.NoError(t, err)
	secondChangeRequestID := secondChangeRequestResponse.ChangeRequest.Metadata.Id

	_, err = actOnCanvasChangeRequestAction(secondUserCtx, r, canvasID, secondChangeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE)
	require.NoError(t, err)
	_, err = actOnCanvasChangeRequestAction(secondUserCtx, r, canvasID, secondChangeRequestID, pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH)
	require.NoError(t, err)

	canvasAfterPublish, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
	require.NoError(t, err)
	liveAfterPublish, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), uuid.MustParse(canvasID))
	require.NoError(t, err)
	require.Len(t, liveAfterPublish.Nodes, 1)
	assert.Equal(t, "Draft Two", liveAfterPublish.Nodes[0].Name)
	_ = canvasAfterPublish

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

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "approve-then-publish")
	createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
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

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "publish-needs-approval")
	createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
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

func TestPublishChangeRequestWithDuplicateNameReturnsAlreadyExists(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	existingCanvasID := createCanvasWithChangeManagement(ctx, t, r, "existing-live-canvas")
	canvasID := createCanvasWithChangeManagement(ctx, t, r, "change-request-duplicate-name")
	draftVersionID := createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	commitDraftMetadataOnly(ctx, t, r, canvasID, draftVersionID, "publish-duplicate-live", "")

	existingCanvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(existingCanvasID))
	require.NoError(t, err)

	require.NoError(t, database.Conn().
		Model(&models.CanvasVersion{}).
		Where("id = ?", *existingCanvas.LiveVersionID).
		Update("name", "publish-duplicate-live").
		Error)
	require.NoError(t, database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", existingCanvas.ID).
		Update("name", "publish-duplicate-live").
		Error)

	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
	require.NoError(t, err)

	_, err = actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		createResponse.ChangeRequest.Metadata.Id,
		pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE,
	)
	require.NoError(t, err)

	_, err = actOnCanvasChangeRequestAction(
		ctx,
		r,
		canvasID,
		createResponse.ChangeRequest.Metadata.Id,
		pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH,
	)
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, grpcstatus.Code(err))

	existingCanvas, err = models.FindCanvas(r.Organization.ID, uuid.MustParse(existingCanvasID))
	require.NoError(t, err)
	assert.Equal(t, "publish-duplicate-live", existingCanvas.Name)
}

func TestUnapproveRequiresReapprovalBeforePublish(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "unapprove-before-publish")
	createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
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

	canvasID := createCanvasWithChangeManagement(ctx, t, r, "reject-invalidates-approvals")
	createDraftVersion(ctx, t, r, canvasID, "Draft Name")
	createResponse, err := CreateCanvasChangeRequest(ctx, r.Organization.ID.String(), canvasID, "")
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
		r.GitProvider,
		r.Organization.ID.String(),
		canvasID,
		changeRequestID,
		action,
		testWebhookBaseURL,
	)
}
