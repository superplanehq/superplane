package factories

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func UpdateFactoryLine(ctx context.Context, organizationID string, req *pb.UpdateFactoryLineRequest) (*pb.UpdateFactoryLineResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	lineID, err := parseLineID(req.GetLineId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	line, err := factory.FindLine(db, lineID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(req.GetName())
		if trimmed == "" {
			return nil, factoryErrorToStatus(invalidArgument("name cannot be empty"), "failed to update factory line")
		}
		name = &trimmed
	}

	var steps []models.FactoryLineStep
	if len(req.GetSteps()) > 0 {
		steps, err = parseLineSteps(db, orgID, factoryID, req.GetSteps())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory line")
		}
	}

	if name == nil && steps == nil {
		return nil, factoryErrorToStatus(invalidArgument("name or steps must be provided"), "failed to update factory line")
	}

	if err := line.Update(db, name, steps); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	if publishErr := messages.PublishFactoryUpdated(factory.ID.String(), "line.updated"); publishErr != nil {
		log.Errorf("failed to publish factory updated RabbitMQ message: %v", publishErr)
	}

	return &pb.UpdateFactoryLineResponse{
		Line: serializeFactoryLine(line),
	}, nil
}
