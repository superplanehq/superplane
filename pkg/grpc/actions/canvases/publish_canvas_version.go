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
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func PublishCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	gitProv gitprovider.Provider,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.PublishCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	organizationUUID := uuid.MustParse(organizationID)
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	publishedVersion, err := publishDraftVersionInTransaction(
		ctx, encryptor, reg, gitProv, organizationID, organizationUUID, canvasUUID, versionUUID, userUUID, authService, webhookBaseURL,
	)
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, err
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), publishedVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.PublishCanvasVersionResponse{
		Version: SerializeCanvasVersion(publishedVersion, organizationID, nil),
	}, nil
}

func publishDraftVersionInTransaction(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	gitProv gitprovider.Provider,
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
				return grpcerrors.NotFound(findErr, "version not found")
			}
			return findErr
		}

		if version.State != models.CanvasVersionStateDraft {
			return grpcerrors.FailedPrecondition(nil, "only draft versions can be published")
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return grpcerrors.PermissionDenied(nil, "version owner mismatch")
		}

		hasStaging, err := models.HasWorkflowStagingInTransaction(tx, version.ID)
		if err != nil {
			return err
		}

		if hasStaging {
			return grpcerrors.FailedPrecondition(nil, "draft version has staged changes")
		}

		nameErr := ensureCanvasNameAvailableInTransaction(tx, organizationUUID, canvasUUID, version.Name)
		if errors.Is(nameErr, models.ErrCanvasNameAlreadyExists) {
			return grpcerrors.AlreadyExists(nil, "Canvas with the same name already exists")
		}
		if nameErr != nil {
			return nameErr
		}

		liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvasUUID)
		if err != nil {
			return err
		}

		err = publishCanvasVersionInTransaction(
			ctx,
			tx,
			liveVersion,
			version,
			changesets.CanvasPublisherOptions{
				Registry:       reg,
				OrgID:          organizationUUID,
				Encryptor:      encryptor,
				AuthService:    authService,
				WebhookBaseURL: webhookBaseURL,
				GitProvider:    gitProv,
			},
		)
		if err != nil {
			log.Errorf("failed to publish canvas version: %v", err)
			return err
		}

		publishedVersion = version
		return nil
	})

	if err != nil {
		return nil, err
	}

	return publishedVersion, nil
}
