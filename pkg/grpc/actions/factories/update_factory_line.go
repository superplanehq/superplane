package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateFactoryLine(ctx context.Context, deps IntakeDependencies, organizationID string, req *pb.UpdateFactoryLineRequest) (*pb.UpdateFactoryLineResponse, error) {
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

	var line *models.FactoryLine
	err = db.Transaction(func(tx *gorm.DB) error {
		if req.StageConfiguration != nil {
			factory, err = models.LockAutomationFactory(tx, orgID, factory.ID)
			if err != nil {
				return err
			}
		}
		line, err = factory.FindLineForUpdate(tx, lineID)
		if err != nil {
			return err
		}

		var name *string
		if req.Name != nil {
			trimmed := strings.TrimSpace(req.GetName())
			if trimmed == "" {
				return invalidArgument("name cannot be empty")
			}
			name = &trimmed
		}

		if req.StageConfiguration != nil && len(req.GetSteps()) > 0 {
			return invalidArgument("steps and stages cannot be updated together")
		}

		var stages []models.FactoryLineStage
		var steps []models.FactoryLineStep
		if len(req.GetSteps()) > 0 {
			steps, err = parseLineSteps(tx, orgID, factory.ID, req.GetSteps())
			if err != nil {
				return err
			}
			stages, err = line.ReconcileStages(tx, steps)
			if err != nil {
				return err
			}
			if len(stages) > factoryLineStageMaxCount {
				return invalidArgument("a line cannot have more than 64 stages")
			}
		}

		var columnColors map[string]string
		if req.ColumnColors != nil {
			columnColors, err = parseLineColumnColors(req.GetColumnColors())
			if err != nil {
				return err
			}
		}

		if req.StageConfiguration != nil {
			organization, findErr := models.FindOrganizationByIDInTransaction(tx, orgID.String())
			if findErr != nil {
				return findErr
			}
			if !organization.HasExperimentalFeature(features.FeatureFactoryCustomAutomations) {
				return errCustomAutomationsDisabled
			}
			stages, err = parseFactoryLineStages(line, req.GetStageConfiguration().GetStages())
			if err != nil {
				return err
			}
			if columnColors == nil {
				columnColors = line.ColumnColorsValue()
			}
			columnColors = remapNewFactoryLineStageColors(req.GetStageConfiguration().GetStages(), stages, columnColors)
			backend := CanvasAutomationBackend{Deps: deps}
			if err := backend.moveAutomationCards(ctx, tx, factory, line, stages, req.StageConfiguration); err != nil {
				return err
			}
			if err := line.DetachRemovedCustomStageAutomations(tx, stages); err != nil {
				return err
			}
		}

		if name == nil && steps == nil && columnColors == nil && stages == nil {
			return invalidArgument("name, steps, column colors, or stages must be provided")
		}

		return line.UpdateConfiguration(tx, name, steps, columnColors, stages)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory line")
	}

	return &pb.UpdateFactoryLineResponse{
		Line: serializeFactoryLine(line),
	}, nil
}
