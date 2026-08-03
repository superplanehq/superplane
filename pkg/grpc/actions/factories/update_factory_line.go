package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
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

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	line, err := models.FindFactoryLine(tx, orgID, factoryID, lineID)
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
		steps, err = parseLineSteps(tx, orgID, factoryID, req.GetSteps())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory line")
		}
	}

	if name == nil && steps == nil {
		return nil, factoryErrorToStatus(invalidArgument("name or steps must be provided"), "failed to update factory line")
	}

	if err := line.Update(tx, name, steps); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	return &pb.UpdateFactoryLineResponse{
		Line: serializeFactoryLine(line),
	}, nil
}
