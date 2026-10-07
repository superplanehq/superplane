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

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, automationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
		return nil, factoryErrorToStatus(gorm.ErrRecordNotFound, "failed to delete factory automation")
	}

	if err := rejectReservedFactoryAutomation(db, factory, canvas); err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	if _, err := canvases.DeleteCanvas(ctx, db, canvas); err != nil {
		return nil, err
	}

	return &pb.DeleteFactoryAutomationResponse{}, nil
}

func rejectReservedFactoryAutomation(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas) error {
	if _, err := models.FindFactoryIntakeByCanvasID(tx, canvas.ID); err == nil {
		return errFactoryAutomationReserved
	} else if !errors.Is(err, models.ErrFactoryIntakeNotFound) {
		return err
	}

	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, canvas.ID)
	if err != nil {
		return err
	}
	if handler != nil {
		return errFactoryAutomationReserved
	}

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}
	if models.IsBacklogFactoryApp(liveVersion.Nodes, liveVersion.Edges) {
		return errFactoryAutomationReserved
	}

	lines, err := factory.ListLines(tx)
	if err != nil {
		return err
	}
	for _, line := range lines {
		for _, step := range line.Steps {
			if step.AppID == canvas.ID {
				return errFactoryAutomationReserved
			}
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
