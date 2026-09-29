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

func UpdateFactory(ctx context.Context, organizationID string, req *pb.UpdateFactoryRequest) (*pb.UpdateFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	if err := factory.Update(db, req.Name, req.Description, req.Key); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	if req.GetClearHostedSpendBudget() {
		if err := factory.UpdateHostedSpendBudget(db, nil); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory")
		}
	} else if req.HostedSpendBudgetCents != nil {
		if err := factory.UpdateHostedSpendBudget(db, req.HostedSpendBudgetCents); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory")
		}
	}

	if req.Planning != nil {
		planning, err := factoryPlanningFromProto(req.Planning, factory.Planning())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory")
		}
		if err := rejectAutoStartLineOutsideFactory(db, factory, planning.AutoStartLineID); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory")
		}
		if err := factory.UpdatePlanning(db, planning); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory")
		}
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	return &pb.UpdateFactoryResponse{
		Factory: serialized,
	}, nil
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
