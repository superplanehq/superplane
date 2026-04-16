package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func PublishCanvasChangeRequest(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	changeRequestID string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*models.CanvasChangeRequest, *models.CanvasVersion, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	organizationUUID := uuid.MustParse(organizationID)
	actorUserUUID := uuid.MustParse(userID)

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	changeManagementEnabled, modeErr := isChangeManagementEnabledForCanvas(canvas)
	if modeErr != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to load change management setting: %v", modeErr)
	}
	if !changeManagementEnabled {
		return nil, nil, status.Error(codes.FailedPrecondition, "change management is disabled for this canvas")
	}

	var version *models.CanvasVersion
	var request *models.CanvasChangeRequest
	var liveVersion *models.CanvasVersion
	var renewedDraftVersion *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		canvasForUpdate, canvasErr := models.FindCanvasInTransaction(tx, organizationUUID, canvasUUID)
		if canvasErr != nil {
			return canvasErr
		}

		request, err = models.FindCanvasChangeRequestInTransaction(tx, canvasUUID, changeRequestUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "change request not found")
			}
			return err
		}

		if request.Status == models.CanvasChangeRequestStatusPublished {
			return status.Error(codes.FailedPrecondition, "change request was already merged")
		}
		if request.Status == models.CanvasChangeRequestStatusRejected {
			return status.Error(codes.FailedPrecondition, "change request is rejected")
		}

		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, request.VersionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		if err := refreshCanvasChangeRequestDiffInTransaction(tx, canvasForUpdate, version, request); err != nil {
			return err
		}
		if len(request.ConflictingNodeIDs) > 0 {
			return status.Error(codes.FailedPrecondition, "change request has conflicts")
		}
		if !isOpenCanvasChangeRequestStatus(request.Status) {
			return status.Error(codes.FailedPrecondition, "change request cannot be published in its current status")
		}
		approvals, approvalsErr := models.ListCanvasChangeRequestApprovalsInTransaction(tx, canvasUUID, request.ID)
		if approvalsErr != nil {
			return approvalsErr
		}
		if publishCheckErr := ensureCanvasChangeRequestReadyToPublish(canvasForUpdate, approvals); publishCheckErr != nil {
			return publishCheckErr
		}

		baseNodes, baseEdges, liveNodes, liveEdges, resolveErr := resolveCanvasChangeRequestBaseAndLiveInTransaction(
			tx,
			canvasForUpdate,
			request,
		)
		if resolveErr != nil {
			return resolveErr
		}

		mergedNodes, mergedEdges := mergeCanvasVersionIntoLive(
			baseNodes,
			baseEdges,
			liveNodes,
			liveEdges,
			version.Nodes,
			version.Edges,
			request.ChangedNodeIDs,
		)

		publisherOwnerID := actorUserUUID
		if request.OwnerID != nil {
			publisherOwnerID = *request.OwnerID
		}

		mergedVersion, createVersionErr := models.CreateCanvasSnapshotVersionInTransaction(
			tx,
			canvasUUID,
			publisherOwnerID,
			mergedNodes,
			mergedEdges,
		)
		if createVersionErr != nil {
			return createVersionErr
		}

		publisher, err := changesets.NewCanvasPublisher(tx, mergedVersion, changesets.CanvasPublisherOptions{
			Registry:       registry,
			OrgID:          organizationUUID,
			Encryptor:      encryptor,
			AuthService:    authService,
			WebhookBaseURL: webhookBaseURL,
		})
		if err != nil {
			return err
		}

		if err := publisher.Publish(ctx); err != nil {
			return err
		}

		liveVersion = mergedVersion
		canvasForUpdate.LiveVersionID = &liveVersion.ID
		canvasForUpdate.UpdatedAt = liveVersion.UpdatedAt

		now := time.Now()
		request.Status = models.CanvasChangeRequestStatusPublished
		request.PublishedAt = &now
		request.UpdatedAt = &now
		if saveErr := tx.Save(request).Error; saveErr != nil {
			return saveErr
		}

		if refreshErr := refreshOpenCanvasChangeRequestsInTransaction(tx, organizationUUID, canvasUUID, request.ID); refreshErr != nil {
			return refreshErr
		}

		if request.OwnerID != nil {
			renewedDraftVersion, err = models.SaveCanvasDraftInTransaction(
				tx,
				canvasUUID,
				*request.OwnerID,
				liveVersion.Nodes,
				liveVersion.Edges,
			)
			if err != nil {
				return err
			}
		}

		canvas = canvasForUpdate
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, nil, err
		}
		return nil, nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if liveVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), liveVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
		}
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}
	if renewedDraftVersion != nil {
		if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), renewedDraftVersion.ID.String()).PublishVersionUpdated(); err != nil {
			log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
		}
	}

	return request, version, nil
}
