package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateConsole(
	ctx context.Context,
	organizationID,
	canvasID string,
	versionID string,
	panels []*pb.Console_Panel,
	layout []*pb.Console_LayoutItem,
) (*pb.UpdateConsoleResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	modelPanels, err := deserializeConsolePanels(panels)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	modelLayout := deserializeConsoleLayout(layout)

	if err := validateConsoleInput(modelPanels, modelLayout); err != nil {
		return nil, err
	}

	var newVersion *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		resolvedVersionID, resolveErr := resolveConsoleVersionID(tx, canvas, strings.TrimSpace(versionID))
		if resolveErr != nil {
			return resolveErr
		}

		version, loadErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, resolvedVersionID)
		if loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return loadErr
		}

		if version.State == models.CanvasVersionStatePublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if _, draftErr := models.FindCanvasDraftByVersionInTransaction(tx, canvas.ID, userUUID, version.ID); draftErr != nil {
			if errors.Is(draftErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "version is not your current edit version")
			}
			return draftErr
		}

		v, updateErr := models.UpdateCanvasVersionConsoleInTransaction(tx, version, modelPanels, modelLayout)
		if updateErr != nil {
			return updateErr
		}

		newVersion = v
		return nil
	})

	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update console")
		return nil, status.Error(codes.Internal, "failed to update console")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), newVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version update RabbitMQ message: %v", err)
	}

	console, err := serializeConsole(newVersion)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize console")
	}

	return &pb.UpdateConsoleResponse{Console: console}, nil
}

func validateConsoleInput(panels []models.ConsolePanel, layout []models.ConsoleLayoutItem) error {
	if err := models.ValidateConsoleContent(panels, layout); err != nil {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return nil
}
