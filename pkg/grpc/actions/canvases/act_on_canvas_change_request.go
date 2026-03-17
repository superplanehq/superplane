package canvases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ActOnCanvasChangeRequest(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	changeRequestID string,
	action pb.ActOnCanvasChangeRequestRequest_Action,
	webhookBaseURL string,
) (*pb.ActOnCanvasChangeRequestResponse, error) {
	if err := validateActOnCanvasChangeRequestAction(action); err != nil {
		return nil, err
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	actorUserID := uuid.MustParse(userID)

	if action == pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH {
		request, version, err := PublishCanvasChangeRequest(
			ctx,
			encryptor,
			registry,
			organizationID,
			canvasID,
			changeRequestID,
			webhookBaseURL,
		)
		if err != nil {
			return nil, err
		}

		return &pb.ActOnCanvasChangeRequestResponse{
			ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
		}, nil
	}

	organizationUUID := uuid.MustParse(organizationID)
	canvasUUID, changeRequestUUID, err := parseActOnCanvasChangeRequestIDs(canvasID, changeRequestID)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}
	if err := validateActOnCanvasChangeRequestCanvas(canvas); err != nil {
		return nil, err
	}

	request, version, err := runActOnCanvasChangeRequestTransaction(
		authService,
		organizationID,
		organizationUUID,
		canvasUUID,
		changeRequestUUID,
		actorUserID,
		action,
	)
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to act on change request: %v", err)
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.ActOnCanvasChangeRequestResponse{
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}

func validateActOnCanvasChangeRequestAction(action pb.ActOnCanvasChangeRequestRequest_Action) error {
	switch action {
	case pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE,
		pb.ActOnCanvasChangeRequestRequest_ACTION_UNAPPROVE,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REJECT,
		pb.ActOnCanvasChangeRequestRequest_ACTION_REOPEN,
		pb.ActOnCanvasChangeRequestRequest_ACTION_PUBLISH:
		return nil
	case pb.ActOnCanvasChangeRequestRequest_ACTION_UNSPECIFIED:
		return status.Error(codes.InvalidArgument, "action is required")
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported action %q", action.String())
	}
}

func parseActOnCanvasChangeRequestIDs(canvasID string, changeRequestID string) (uuid.UUID, uuid.UUID, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
	}

	return canvasUUID, changeRequestUUID, nil
}

func validateActOnCanvasChangeRequestCanvas(canvas *models.Canvas) error {
	if canvas.IsTemplate {
		return status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	versioningEnabled, modeErr := isCanvasVersioningEnabledForCanvas(canvas)
	if modeErr != nil {
		return status.Errorf(codes.Internal, "failed to load canvas versioning: %v", modeErr)
	}
	if !versioningEnabled {
		return status.Error(codes.FailedPrecondition, "canvas versioning is disabled for this canvas")
	}

	return nil
}

func runActOnCanvasChangeRequestTransaction(
	authService authorization.Authorization,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	changeRequestUUID uuid.UUID,
	actorUserID uuid.UUID,
	action pb.ActOnCanvasChangeRequestRequest_Action,
) (*models.CanvasChangeRequest, *models.CanvasVersion, error) {
	var request *models.CanvasChangeRequest
	var version *models.CanvasVersion

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasInTx, err := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if err != nil {
			return err
		}

		request, version, err = findActOnCanvasChangeRequestModelsInTransaction(tx, canvasUUID, changeRequestUUID)
		if err != nil {
			return err
		}

		approvals, err := models.ListCanvasChangeRequestApprovalsInTransaction(tx, canvasUUID, request.ID)
		if err != nil {
			return err
		}

		return applyActOnCanvasChangeRequestActionInTransaction(
			tx,
			authService,
			organizationID,
			canvasInTx,
			request,
			version,
			approvals,
			actorUserID,
			action,
		)
	})
	if err != nil {
		return nil, nil, err
	}

	return request, version, nil
}

func findActOnCanvasChangeRequestModelsInTransaction(
	tx *gorm.DB,
	canvasUUID uuid.UUID,
	changeRequestUUID uuid.UUID,
) (*models.CanvasChangeRequest, *models.CanvasVersion, error) {
	request, err := models.FindCanvasChangeRequestInTransaction(tx, canvasUUID, changeRequestUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, status.Error(codes.NotFound, "change request not found")
		}
		return nil, nil, err
	}

	version, err := models.FindCanvasVersionInTransaction(tx, canvasUUID, request.VersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, status.Error(codes.NotFound, "change request version not found")
		}
		return nil, nil, err
	}

	return request, version, nil
}

func applyActOnCanvasChangeRequestActionInTransaction(
	tx *gorm.DB,
	authService authorization.Authorization,
	organizationID string,
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
	version *models.CanvasVersion,
	approvals []models.CanvasChangeRequestApproval,
	actorUserID uuid.UUID,
	action pb.ActOnCanvasChangeRequestRequest_Action,
) error {
	switch action {
	case pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE:
		return approveCanvasChangeRequestInTransaction(
			tx,
			authService,
			organizationID,
			canvas,
			request,
			approvals,
			actorUserID,
		)
	case pb.ActOnCanvasChangeRequestRequest_ACTION_UNAPPROVE:
		return unapproveCanvasChangeRequestInTransaction(tx, canvas, request, approvals, actorUserID)
	case pb.ActOnCanvasChangeRequestRequest_ACTION_REJECT:
		return rejectCanvasChangeRequestInTransaction(
			tx,
			authService,
			organizationID,
			canvas,
			request,
			approvals,
			actorUserID,
		)
	case pb.ActOnCanvasChangeRequestRequest_ACTION_REOPEN:
		return reopenCanvasChangeRequestInTransaction(tx, canvas, request, version)
	default:
		return status.Error(codes.InvalidArgument, "unsupported action")
	}
}

func approveCanvasChangeRequestInTransaction(
	tx *gorm.DB,
	authService authorization.Authorization,
	organizationID string,
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
	approvals []models.CanvasChangeRequestApproval,
	actorUserID uuid.UUID,
) error {
	if request.Status == models.CanvasChangeRequestStatusPublished {
		return status.Error(codes.FailedPrecondition, "published change requests cannot be approved")
	}
	if request.Status == models.CanvasChangeRequestStatusRejected {
		return status.Error(codes.FailedPrecondition, "rejected change requests must be reopened before approval")
	}
	if !isOpenCanvasChangeRequestStatus(request.Status) {
		return status.Error(codes.FailedPrecondition, "only open change requests can be approved")
	}
	if request.IsConflicted() {
		return status.Error(codes.FailedPrecondition, "conflicted change requests cannot be approved")
	}

	approverIndex, approver, err := resolveActingApprover(
		authService,
		organizationID,
		canvas.EffectiveChangeRequestApprovers(),
		approvals,
		actorUserID,
		false,
	)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := invalidateApprovalIndexInTransaction(tx, canvas.ID, request.ID, approverIndex, now); err != nil {
		return err
	}
	if err := createCanvasChangeRequestApprovalInTransaction(
		tx,
		canvas.ID,
		request.ID,
		approverIndex,
		approver,
		actorUserID,
		models.CanvasChangeRequestApprovalStateApproved,
		now,
	); err != nil {
		return err
	}

	request.UpdatedAt = &now
	return tx.Save(request).Error
}

func unapproveCanvasChangeRequestInTransaction(
	tx *gorm.DB,
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
	approvals []models.CanvasChangeRequestApproval,
	actorUserID uuid.UUID,
) error {
	if request.Status == models.CanvasChangeRequestStatusPublished {
		return status.Error(codes.FailedPrecondition, "published change requests cannot be unapproved")
	}
	if request.Status == models.CanvasChangeRequestStatusRejected {
		return status.Error(codes.FailedPrecondition, "rejected change requests cannot be unapproved")
	}
	if !isOpenCanvasChangeRequestStatus(request.Status) {
		return status.Error(codes.FailedPrecondition, "only open change requests can be unapproved")
	}

	currentApproval := findActorActiveApproval(approvals, actorUserID)
	if currentApproval == nil {
		return status.Error(codes.FailedPrecondition, "you do not have an active approval to remove")
	}

	now := time.Now()
	if err := invalidateApprovalIndexInTransaction(tx, canvas.ID, request.ID, currentApproval.ApproverIndex, now); err != nil {
		return err
	}

	approver := models.CanvasChangeRequestApprover{
		Type: currentApproval.ApproverType,
	}
	if currentApproval.ApproverUserID != nil {
		approver.User = currentApproval.ApproverUserID.String()
	}
	if currentApproval.ApproverRole != nil {
		approver.Role = *currentApproval.ApproverRole
	}

	if err := createCanvasChangeRequestApprovalInTransaction(
		tx,
		canvas.ID,
		request.ID,
		currentApproval.ApproverIndex,
		approver,
		actorUserID,
		models.CanvasChangeRequestApprovalStateUnapproved,
		now,
	); err != nil {
		return err
	}

	request.UpdatedAt = &now
	return tx.Save(request).Error
}

func rejectCanvasChangeRequestInTransaction(
	tx *gorm.DB,
	authService authorization.Authorization,
	organizationID string,
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
	approvals []models.CanvasChangeRequestApproval,
	actorUserID uuid.UUID,
) error {
	if request.Status == models.CanvasChangeRequestStatusPublished {
		return status.Error(codes.FailedPrecondition, "published change requests cannot be rejected")
	}
	if request.Status == models.CanvasChangeRequestStatusRejected {
		return nil
	}
	if !isOpenCanvasChangeRequestStatus(request.Status) {
		return status.Error(codes.FailedPrecondition, "only open change requests can be rejected")
	}

	approverIndex, approver, err := resolveActingApprover(
		authService,
		organizationID,
		canvas.EffectiveChangeRequestApprovers(),
		approvals,
		actorUserID,
		true,
	)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := models.InvalidateCanvasChangeRequestApprovalsInTransaction(tx, canvas.ID, request.ID, now); err != nil {
		return err
	}
	if err := createCanvasChangeRequestApprovalInTransaction(
		tx,
		canvas.ID,
		request.ID,
		approverIndex,
		approver,
		actorUserID,
		models.CanvasChangeRequestApprovalStateRejected,
		now,
	); err != nil {
		return err
	}

	request.Status = models.CanvasChangeRequestStatusRejected
	request.UpdatedAt = &now
	return tx.Save(request).Error
}

func reopenCanvasChangeRequestInTransaction(
	tx *gorm.DB,
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
	version *models.CanvasVersion,
) error {
	if request.Status != models.CanvasChangeRequestStatusRejected {
		return status.Error(codes.FailedPrecondition, "only rejected change requests can be reopened")
	}

	baseNodes, baseEdges, liveNodes, liveEdges, err := resolveCanvasChangeRequestBaseAndLiveInTransaction(tx, canvas, request)
	if err != nil {
		return err
	}

	diff := computeCanvasChangeRequestDiff(baseNodes, baseEdges, liveNodes, liveEdges, version.Nodes, version.Edges)
	now := time.Now()
	if err := models.InvalidateCanvasChangeRequestApprovalsInTransaction(tx, canvas.ID, request.ID, now); err != nil {
		return err
	}

	request.ChangedNodeIDs = datatypes.NewJSONSlice(diff.ChangedNodeIDs)
	request.ConflictingNodeIDs = datatypes.NewJSONSlice(diff.ConflictingNodeIDs)
	request.UpdatedAt = &now
	request.Status = models.CanvasChangeRequestStatusOpen

	return tx.Save(request).Error
}

func resolveActingApprover(
	authService authorization.Authorization,
	organizationID string,
	approvers []models.CanvasChangeRequestApprover,
	approvals []models.CanvasChangeRequestApproval,
	actorUserID uuid.UUID,
	allowAlreadyApproved bool,
) (int, models.CanvasChangeRequestApprover, error) {
	roles, err := authService.GetUserRolesForOrg(actorUserID.String(), organizationID)
	if err != nil {
		return -1, models.CanvasChangeRequestApprover{}, status.Errorf(codes.Internal, "failed to resolve user roles: %v", err)
	}

	roleSet := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if role == nil {
			continue
		}
		roleSet[role.Name] = struct{}{}
	}

	activeByIndex := activeApprovalsByIndex(approvals)
	foundEligible := false
	firstEligibleIndex := -1
	var firstEligibleApprover models.CanvasChangeRequestApprover
	for index, approver := range approvers {
		if !actorMatchesApprover(actorUserID, roleSet, approver) {
			continue
		}
		foundEligible = true
		if firstEligibleIndex == -1 {
			firstEligibleIndex = index
			firstEligibleApprover = approver
		}

		active := activeByIndex[index]
		if active == nil {
			return index, approver, nil
		}

		if active.State == models.CanvasChangeRequestApprovalStateApproved {
			if allowAlreadyApproved {
				continue
			}
			if active.ActorUserID != nil && *active.ActorUserID == actorUserID {
				continue
			}
			continue
		}

		if active.State == models.CanvasChangeRequestApprovalStateRejected {
			if allowAlreadyApproved {
				continue
			}
		}

		return index, approver, nil
	}

	if !foundEligible {
		return -1, models.CanvasChangeRequestApprover{}, status.Error(codes.PermissionDenied, "you are not allowed to act as an approver for this change request")
	}

	if allowAlreadyApproved {
		return firstEligibleIndex, firstEligibleApprover, nil
	}

	return -1, models.CanvasChangeRequestApprover{}, status.Error(codes.FailedPrecondition, "all of your eligible approvals are already completed")
}

func actorMatchesApprover(
	actorUserID uuid.UUID,
	actorRoles map[string]struct{},
	approver models.CanvasChangeRequestApprover,
) bool {
	switch approver.Type {
	case models.CanvasChangeRequestApproverTypeAnyone:
		return true
	case models.CanvasChangeRequestApproverTypeUser:
		return approver.User == actorUserID.String()
	case models.CanvasChangeRequestApproverTypeRole:
		_, ok := actorRoles[approver.Role]
		return ok
	default:
		return false
	}
}

func activeApprovalsByIndex(approvals []models.CanvasChangeRequestApproval) map[int]*models.CanvasChangeRequestApproval {
	result := make(map[int]*models.CanvasChangeRequestApproval)
	for i := range approvals {
		approval := approvals[i]
		if approval.InvalidatedAt != nil {
			continue
		}

		existing := result[approval.ApproverIndex]
		if existing == nil {
			approvalCopy := approval
			result[approval.ApproverIndex] = &approvalCopy
			continue
		}

		existingCreatedAt := time.Time{}
		if existing.CreatedAt != nil {
			existingCreatedAt = *existing.CreatedAt
		}
		approvalCreatedAt := time.Time{}
		if approval.CreatedAt != nil {
			approvalCreatedAt = *approval.CreatedAt
		}
		if !approvalCreatedAt.After(existingCreatedAt) {
			continue
		}

		approvalCopy := approval
		result[approval.ApproverIndex] = &approvalCopy
	}

	return result
}

func findActorActiveApproval(
	approvals []models.CanvasChangeRequestApproval,
	actorUserID uuid.UUID,
) *models.CanvasChangeRequestApproval {
	var selected *models.CanvasChangeRequestApproval
	for i := range approvals {
		approval := approvals[i]
		if approval.InvalidatedAt != nil {
			continue
		}
		if approval.State != models.CanvasChangeRequestApprovalStateApproved {
			continue
		}
		if approval.ActorUserID == nil || *approval.ActorUserID != actorUserID {
			continue
		}

		if selected == nil {
			approvalCopy := approval
			selected = &approvalCopy
			continue
		}

		selectedCreatedAt := time.Time{}
		if selected.CreatedAt != nil {
			selectedCreatedAt = *selected.CreatedAt
		}
		approvalCreatedAt := time.Time{}
		if approval.CreatedAt != nil {
			approvalCreatedAt = *approval.CreatedAt
		}
		if !approvalCreatedAt.After(selectedCreatedAt) {
			continue
		}

		approvalCopy := approval
		selected = &approvalCopy
	}

	return selected
}

func invalidateApprovalIndexInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	changeRequestID uuid.UUID,
	approverIndex int,
	now time.Time,
) error {
	return tx.
		Model(&models.CanvasChangeRequestApproval{}).
		Where("workflow_id = ?", workflowID).
		Where("workflow_change_request_id = ?", changeRequestID).
		Where("approver_index = ?", approverIndex).
		Where("invalidated_at IS NULL").
		Updates(map[string]any{
			"invalidated_at": now,
			"updated_at":     now,
		}).
		Error
}

func createCanvasChangeRequestApprovalInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	changeRequestID uuid.UUID,
	approverIndex int,
	approver models.CanvasChangeRequestApprover,
	actorUserID uuid.UUID,
	state string,
	now time.Time,
) error {
	approval := &models.CanvasChangeRequestApproval{
		ID:                      uuid.New(),
		WorkflowID:              workflowID,
		WorkflowChangeRequestID: changeRequestID,
		ApproverIndex:           approverIndex,
		ApproverType:            approver.Type,
		ActorUserID:             &actorUserID,
		State:                   state,
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	if approver.Type == models.CanvasChangeRequestApproverTypeUser && approver.User != "" {
		approverUserID, parseErr := uuid.Parse(approver.User)
		if parseErr != nil {
			return status.Errorf(codes.Internal, "invalid approver user id: %v", parseErr)
		}
		approval.ApproverUserID = &approverUserID
	}
	if approver.Type == models.CanvasChangeRequestApproverTypeRole && approver.Role != "" {
		role := approver.Role
		approval.ApproverRole = &role
	}

	if err := models.CreateCanvasChangeRequestApprovalInTransaction(tx, approval); err != nil {
		return fmt.Errorf("failed to create change request approval record: %w", err)
	}

	return nil
}
