package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func AddSource(ctx context.Context, organizationID string, req *pb.AddSourceRequest) (*pb.AddSourceResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add factory source")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add factory source")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to add factory source")
	}

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to add factory source")
	}

	integrationID, err := resolveIntegrationRef(tx, orgID, req.GetIntegration())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add factory source")
	}

	source, err := models.CreateFactorySource(
		tx,
		orgID,
		factoryID,
		integrationID,
		name,
		structToMap(req.GetConfiguration()),
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add factory source")
	}

	protoSource, err := serializeFactorySource(tx, orgID, source)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add factory source")
	}

	return &pb.AddSourceResponse{
		Source: protoSource,
	}, nil
}
