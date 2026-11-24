package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DeleteFactoryAutomation(
	ctx context.Context,
	organizationID string,
	req *pb.DeleteFactoryAutomationRequest,
) (*pb.DeleteFactoryAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	automationID, err := parseFactoryAutomationID(req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	var canvas *models.Canvas
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := findFactory(tx, orgID, req.GetFactoryId())
		if err != nil {
			return err
		}
		factory, err = models.LockAutomationFactory(tx, orgID, factory.ID)
		if err != nil {
			return err
		}
		canvas, err = models.FindCanvasInTransaction(tx, orgID, automationID)
		if err != nil {
			return err
		}
		if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
			return gorm.ErrRecordNotFound
		}
		if err := detachFactoryAutomation(tx, factory, canvas); err != nil {
			return err
		}
		return canvases.SoftDeleteCanvas(tx, canvas)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}
	canvases.PublishCanvasDeleted(canvas)

	return &pb.DeleteFactoryAutomationResponse{}, nil
}

func detachFactoryAutomation(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas) error {
	if intake, err := models.FindFactoryIntakeByCanvasID(tx, canvas.ID); err == nil {
		if err := intake.Delete(tx); err != nil {
			return err
		}
	} else if !errors.Is(err, models.ErrFactoryIntakeNotFound) {
		return err
	}

	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, canvas.ID)
	if err != nil {
		return err
	}
	if handler != nil {
		if err := handler.Delete(tx); err != nil {
			return err
		}
	}

	lines, err := factory.ListLines(tx)
	if err != nil {
		return err
	}
	for _, line := range lines {
		locked, err := factory.FindLineForUpdate(tx, line.ID)
		if err != nil {
			return err
		}
		if err := locked.DetachAutomation(tx, canvas.ID); err != nil {
			return err
		}
	}

	return nil
}

func parseFactoryAutomationID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid factory automation ID")
	}
	return id, nil
}
