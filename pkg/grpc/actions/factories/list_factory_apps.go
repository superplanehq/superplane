package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func ListFactoryApps(ctx context.Context, organizationID string, req *pb.ListFactoryAppsRequest) (*pb.ListFactoryAppsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	intakes, err := loadFactoryIntakes(db, canvases)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	return &pb.ListFactoryAppsResponse{
		Apps: serializeFactoryApps(canvases, intakes),
	}, nil
}

func loadFactoryIntakes(db *gorm.DB, canvases []models.Canvas) (map[uuid.UUID]factoryIntake, error) {
	intakes := make(map[uuid.UUID]factoryIntake)
	for i := range canvases {
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, &canvases[i])
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		intake, ok := detectFactoryIntake(version)
		if ok {
			intakes[canvases[i].ID] = intake
		}
	}

	return intakes, nil
}
