package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func UpdateFactoryLine(ctx context.Context, organizationID string, req *pb.UpdateFactoryLineRequest) (*pb.UpdateFactoryLineResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	lineID, err := parseLineID(req.GetLineId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}
	factoryID := factory.ID

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

	if req.StageConfiguration != nil && len(req.GetSteps()) > 0 {
		return nil, factoryErrorToStatus(invalidArgument("steps and stages cannot be updated together"), "failed to update factory line")
	}

	var steps []models.FactoryLineStep
	if len(req.GetSteps()) > 0 {
		steps, err = parseLineSteps(db, orgID, factoryID, req.GetSteps())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory line")
		}
	}

	var columnColors map[string]string
	if req.ColumnColors != nil {
		columnColors, err = parseLineColumnColors(req.GetColumnColors())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory line")
		}
	}

	var stages []models.FactoryLineStage
	if req.StageConfiguration != nil {
		organization, findErr := models.FindOrganizationByIDInTransaction(db, orgID.String())
		if findErr != nil {
			return nil, factoryErrorToStatus(findErr, "failed to update factory line")
		}
		if !organization.HasExperimentalFeature(features.FeatureFactoryCustomStages) {
			return nil, factoryErrorToStatus(errFactoryCustomStagesDisabled, "failed to update factory line")
		}
		stages, err = parseFactoryLineStages(db, line, req.GetStageConfiguration().GetStages())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory line")
		}
		if columnColors == nil {
			columnColors = line.ColumnColorsValue()
		}
		columnColors = remapNewFactoryLineStageColors(req.GetStageConfiguration().GetStages(), stages, columnColors)
	}

	if name == nil && steps == nil && columnColors == nil && stages == nil {
		return nil, factoryErrorToStatus(invalidArgument("name, steps, column colors, or stages must be provided"), "failed to update factory line")
	}

	if err := line.UpdateConfiguration(db, name, steps, columnColors, stages); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	return &pb.UpdateFactoryLineResponse{
		Line: serializeFactoryLine(line),
	}, nil
}
