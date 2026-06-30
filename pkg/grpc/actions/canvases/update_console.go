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
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func UpdateConsole(
	ctx context.Context,
	organizationID,
	canvasID string,
	versionID string,
	modelPanels []models.ConsolePanel,
	modelLayout []models.ConsoleLayoutItem,
	discardStaging bool,
) (*models.CanvasVersion, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid canvas_id")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

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
				return grpcerrors.NotFound(loadErr, "version not found")
			}
			return loadErr
		}

		if err := ensureVersionIsOwnedRegisteredDraft(userUUID, version); err != nil {
			return err
		}

		v, updateErr := models.UpdateCanvasVersionConsoleInTransaction(tx, version, modelPanels, modelLayout)
		if updateErr != nil {
			return updateErr
		}

		newVersion = v

		if discardStaging {
			branch, branchErr := models.FindWorkflowBranch(tx, canvas.ID, version.GitBranch)
			if branchErr != nil {
				return branchErr
			}
			return models.DiscardWorkflowStagingInTransaction(tx, branch.ID, userUUID, nil)
		}

		return nil
	})

	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update console")
		return nil, grpcerrors.Internal(err, "failed to update console")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), newVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version update RabbitMQ message: %v", err)
	}

	return newVersion, nil
}

func validateConsoleInput(panels []models.ConsolePanel, layout []models.ConsoleLayoutItem) error {
	if err := models.ValidateConsoleContent(panels, layout); err != nil {
		return grpcerrors.InvalidArgument(err, "invalid request")
	}
	return nil
}
