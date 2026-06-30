package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func DeleteCanvasVersion(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*pb.DeleteCanvasVersionResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, findErr := models.FindCanvasVersionForUpdateInTransaction(tx, canvasUUID, versionUUID)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(findErr, "version not found")
			}
			return findErr
		}

		if version.GitBranch == models.CanvasGitBranchMain {
			return grpcerrors.FailedPrecondition(nil, "cannot delete main branch commits directly")
		}

		return models.DeleteWorkflowBranch(tx, canvasUUID, version.GitBranch)
	})
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to delete canvas branch")
		return nil, grpcerrors.Internal(err, "failed to delete canvas branch")
	}

	if err := messages.PublishVersionDeleted(canvas.ID.String(), versionUUID.String()); err != nil {
		log.Errorf("failed to publish canvas version deleted RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasVersionResponse{}, nil
}
