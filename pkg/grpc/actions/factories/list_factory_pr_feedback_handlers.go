package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func ListFactoryPRFeedbackHandlers(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryPRFeedbackHandlersRequest,
) (*pb.ListFactoryPRFeedbackHandlersResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handlers")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handlers")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handlers")
	}

	handlers, err := factory.ListPRFeedbackHandlers(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handlers")
	}

	specs, err := prFeedbackCanvasSpecs(db, handlers)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory PR feedback handlers")
	}

	return &pb.ListFactoryPRFeedbackHandlersResponse{
		Handlers: serializeFactoryPRFeedbackHandlers(db, orgID, handlers, specs),
	}, nil
}

func prFeedbackCanvasSpecs(db *gorm.DB, handlers []models.FactoryPRFeedbackHandler) (map[uuid.UUID]models.LiveCanvasSpec, error) {
	canvasIDs := make([]uuid.UUID, len(handlers))
	for i := range handlers {
		canvasIDs[i] = handlers[i].CanvasID
	}

	return models.FindLiveCanvasSpecsByCanvasIDs(db, canvasIDs)
}
