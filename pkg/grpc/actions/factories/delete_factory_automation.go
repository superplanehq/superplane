package factories

import (
	"context"

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

	if _, err := canvases.DeleteCanvas(ctx, db, canvas); err != nil {
		return nil, err
	}

	return &pb.DeleteFactoryAutomationResponse{}, nil
}

func parseFactoryAutomationID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid factory automation ID")
	}
	return id, nil
}
