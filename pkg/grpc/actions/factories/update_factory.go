package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateFactory(ctx context.Context, organizationID string, req *pb.UpdateFactoryRequest) (*pb.UpdateFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	var response *pb.UpdateFactoryResponse
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := findFactory(tx, orgID, req.GetId())
		if err != nil {
			return err
		}
		if err := applyFactoryUpdate(tx, factory, req); err != nil {
			return err
		}
		lines, err := factory.ListLines(tx)
		if err != nil {
			return err
		}
		serialized, err := serializeFactoryWithLineMetrics(tx, factory, lines)
		if err != nil {
			return err
		}
		response = &pb.UpdateFactoryResponse{Factory: serialized}
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}
	return response, nil
}

func applyFactoryUpdate(tx *gorm.DB, factory *models.Factory, req *pb.UpdateFactoryRequest) error {
	if err := rejectRequestedFactoryDefault(tx, factory, req); err != nil {
		return err
	}
	if err := factory.Update(tx, req.Name, req.Description, req.Key); err != nil {
		return err
	}

	if req.GetClearHostedSpendBudget() {
		if err := factory.UpdateHostedSpendBudget(tx, nil); err != nil {
			return err
		}
	} else if req.HostedSpendBudgetCents != nil {
		if err := factory.UpdateHostedSpendBudget(tx, req.HostedSpendBudgetCents); err != nil {
			return err
		}
	}

	if req.Planning != nil {
		planning, err := factoryPlanningFromProto(req.Planning)
		if err != nil {
			return err
		}
		if err := rejectAutoStartLineOutsideFactory(tx, factory, planning.AutoStartLineID); err != nil {
			return err
		}
		if err := factory.UpdatePlanning(tx, planning); err != nil {
			return err
		}
	}

	if req.PublicBadgeEnabled != nil {
		if err := factory.UpdatePublicBadgeEnabled(tx, req.GetPublicBadgeEnabled()); err != nil {
			return err
		}
	}
	if req.PublicBadgeShowCost != nil {
		if err := factory.UpdatePublicBadgeShowCost(tx, req.GetPublicBadgeShowCost()); err != nil {
			return err
		}
	}

	if req.GetClearDefaultAgentModel() {
		return factory.UpdateDefaultAgentModel(tx, "", "")
	}
	if req.DefaultAgentModel != nil {
		return setFactoryDefaultAgentModel(tx, factory, req.GetDefaultAgentModel(), req.GetDefaultAgentModelProvider())
	}
	return nil
}

func rejectRequestedFactoryDefault(tx *gorm.DB, factory *models.Factory, req *pb.UpdateFactoryRequest) error {
	if req.GetClearDefaultAgentModel() || req.DefaultAgentModel == nil {
		return nil
	}
	model := strings.TrimSpace(req.GetDefaultAgentModel())
	if model == "" {
		return invalidArgument("default model is required")
	}
	return rejectUnusableFactoryDefault(tx, factory, model, req.GetDefaultAgentModelProvider())
}

func rejectAutoStartLineOutsideFactory(db *gorm.DB, factory *models.Factory, lineID *uuid.UUID) error {
	if lineID == nil {
		return nil
	}
	if _, err := factory.FindLine(db, *lineID); err != nil {
		if errors.Is(err, models.ErrFactoryLineNotFound) {
			return invalidArgument("auto_start_line_id is not a line in this workspace")
		}
		return err
	}
	return nil
}
