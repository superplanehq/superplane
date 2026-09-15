package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CreateFactory(ctx context.Context, organizationID string, req *pb.CreateFactoryRequest) (*pb.CreateFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to create factory")
	}

	// An empty key means the caller did not choose one — older clients and
	// the CLI take this path. CreateFactory then derives a key from the
	// name and walks to a free variant, which a pre-derived key here would
	// skip and turn into a spurious key-already-exists error.
	key := models.NormalizeFactoryKey(req.GetKey())

	db := database.DB(ctx)
	factory, err := models.CreateFactory(db, orgID, name, req.GetDescription(), key)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory")
	}

	serialized := serializeFactory(factory)
	if err := attachFactoryMaxParallelTasks(db, orgID, serialized); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory")
	}

	return &pb.CreateFactoryResponse{
		Factory: serialized,
	}, nil
}
