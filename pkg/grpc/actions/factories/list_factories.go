package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactories(ctx context.Context, organizationID string) (*pb.ListFactoriesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factories")
	}

	db := database.DB(ctx)
	factories, err := models.ListFactories(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factories")
	}

	factoryIDs := make([]uuid.UUID, len(factories))
	for i := range factories {
		factoryIDs[i] = factories[i].ID
	}

	lines, err := models.ListFactoryLinesByFactoryIDs(db, orgID, factoryIDs)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factories")
	}

	return &pb.ListFactoriesResponse{
		Factories: serializeFactories(factories, groupFactoryLinesByFactoryID(lines)),
	}, nil
}

func groupFactoryLinesByFactoryID(lines []models.FactoryLine) map[uuid.UUID][]models.FactoryLine {
	grouped := make(map[uuid.UUID][]models.FactoryLine)
	for i := range lines {
		factoryID := lines[i].FactoryID
		grouped[factoryID] = append(grouped[factoryID], lines[i])
	}
	return grouped
}
