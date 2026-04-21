package canvases

import (
	"context"
	"errors"

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
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func PublishCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.PublishCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	organizationUUID := uuid.MustParse(organizationID)
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	publishedVersion, err := publishDraftVersionInTransaction(
		ctx, encryptor, reg, organizationID, organizationUUID, canvasUUID, versionUUID, userUUID, authService, webhookBaseURL,
	)
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), publishedVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.PublishCanvasVersionResponse{
		Version: SerializeCanvasVersion(publishedVersion, organizationID),
	}, nil
}

func publishDraftVersionInTransaction(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	versionUUID uuid.UUID,
	userUUID uuid.UUID,
	authService authorization.Authorization,
	webhookBaseURL string,
) (*models.CanvasVersion, error) {
	var publishedVersion *models.CanvasVersion

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		version, findErr := models.FindCanvasVersionForUpdateInTransaction(tx, canvasUUID, versionUUID)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return findErr
		}

		if version.State != models.CanvasVersionStateDraft {
			return status.Error(codes.FailedPrecondition, "only draft versions can be published")
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvasUUID)
		if err != nil {
			return err
		}

		publisher, err := changesets.NewCanvasPublisher(tx, version, liveVersion, changesets.CanvasPublisherOptions{
			Registry:       reg,
			OrgID:          organizationUUID,
			Encryptor:      encryptor,
			AuthService:    authService,
			WebhookBaseURL: webhookBaseURL,
		})

		if err != nil {
			log.Errorf("failed to create canvas publisher: %v", err)
			return err
		}

		err = publisher.Publish(ctx)
		if err != nil {
			log.Errorf("failed to publish canvas version: %v", err)
			return err
		}

		refreshErr := refreshOpenCanvasChangeRequestsInTransaction(tx, organizationUUID, canvasUUID, uuid.Nil)
		if refreshErr != nil {
			log.Errorf("failed to refresh open canvas change requests: %v", refreshErr)
			return refreshErr
		}

		publishedVersion = version
		return nil
	})

	if err != nil {
		return nil, err
	}

	return publishedVersion, nil
}
