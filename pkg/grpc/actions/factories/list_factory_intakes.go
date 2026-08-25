package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func ListFactoryIntakes(ctx context.Context, organizationID string, req *pb.ListFactoryIntakesRequest) (*pb.ListFactoryIntakesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intakes")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intakes")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intakes")
	}

	intakes, err := factory.ListIntakes(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intakes")
	}

	specs, err := intakeCanvasSpecs(db, intakes)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory intakes")
	}

	return &pb.ListFactoryIntakesResponse{
		Intakes: serializeFactoryIntakes(intakes, specs),
	}, nil
}

// intakeCanvasSpecs loads the live graph of every intake in one query. Node
// identity is derived from the graph, so listing intakes must not fall back to
// a query per intake.
func intakeCanvasSpecs(db *gorm.DB, intakes []models.FactoryIntake) (map[uuid.UUID]models.LiveCanvasSpec, error) {
	canvasIDs := make([]uuid.UUID, len(intakes))
	for i := range intakes {
		canvasIDs[i] = intakes[i].CanvasID
	}

	return models.FindLiveCanvasSpecsByCanvasIDs(db, canvasIDs)
}
