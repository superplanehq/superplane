package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
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

	if action == pb.ActOnCanvasChangeRequestRequest_ACTION_APPROVE {
		request, version, err := publishCanvasChangeRequestFromAction(
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

	if err := requireActOnCanvasChangeRequestUser(ctx); err != nil {
		return nil, err
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

	request, version, err := runActOnCanvasChangeRequestTransaction(organizationUUID, canvasUUID, changeRequestUUID, action)
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
	if action == pb.ActOnCanvasChangeRequestRequest_ACTION_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "action is required")
	}

	return nil
}

func publishCanvasChangeRequestFromAction(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	changeRequestID string,
	webhookBaseURL string,
) (*models.CanvasChangeRequest, *models.CanvasVersion, error) {
	return PublishCanvasChangeRequest(
		ctx,
		encryptor,
		registry,
		organizationID,
		canvasID,
		changeRequestID,
		webhookBaseURL,
	)
}

func requireActOnCanvasChangeRequestUser(ctx context.Context) error {
	if _, ok := authentication.GetUserIdFromMetadata(ctx); !ok {
		return status.Error(codes.Unauthenticated, "user not authenticated")
	}

	return nil
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
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	changeRequestUUID uuid.UUID,
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

		return applyActOnCanvasChangeRequestActionInTransaction(tx, canvasInTx, request, version, action)
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
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
	version *models.CanvasVersion,
	action pb.ActOnCanvasChangeRequestRequest_Action,
) error {
	switch action {
	case pb.ActOnCanvasChangeRequestRequest_ACTION_REJECT:
		return rejectCanvasChangeRequestInTransaction(tx, request)
	case pb.ActOnCanvasChangeRequestRequest_ACTION_REOPEN:
		return reopenCanvasChangeRequestInTransaction(tx, canvas, request, version)
	default:
		return status.Error(codes.InvalidArgument, "unsupported action")
	}
}

func rejectCanvasChangeRequestInTransaction(tx *gorm.DB, request *models.CanvasChangeRequest) error {
	if request.Status == models.CanvasChangeRequestStatusPublished {
		return status.Error(codes.FailedPrecondition, "published change requests cannot be rejected")
	}
	if request.Status == models.CanvasChangeRequestStatusRejected {
		return nil
	}
	if !isOpenCanvasChangeRequestStatus(request.Status) {
		return status.Error(codes.FailedPrecondition, "only open change requests can be rejected")
	}

	now := time.Now()
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
	request.ChangedNodeIDs = datatypes.NewJSONSlice(diff.ChangedNodeIDs)
	request.ConflictingNodeIDs = datatypes.NewJSONSlice(diff.ConflictingNodeIDs)
	request.UpdatedAt = &now
	request.Status = models.CanvasChangeRequestStatusOpen

	return tx.Save(request).Error
}
