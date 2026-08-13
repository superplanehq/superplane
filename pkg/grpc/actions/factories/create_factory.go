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

	// Fall back to a name-derived key when the client did not send one.
	// The UI usually pre-fills the key from the name; older clients and
	// the CLI still work because we generate the same default here.
	key := models.NormalizeFactoryKey(req.GetKey())
	if key == "" {
		key = models.GenerateFactoryKeyFromName(name)
	}

	factory, err := models.CreateFactory(database.DB(ctx), orgID, name, req.GetDescription(), key)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory")
	}

	return &pb.CreateFactoryResponse{
		Factory: serializeFactory(factory),
	}, nil
}
