package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListWorkOrderChecks(
	ctx context.Context,
	organizationID string,
	req *pb.ListWorkOrderChecksRequest,
) (*pb.ListWorkOrderChecksResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	db := database.DB(ctx)
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	order, err := factoryModel.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	checks, err := order.ListChecks(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	serialized, err := serializeChecks(checks)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order checks")
	}

	return &pb.ListWorkOrderChecksResponse{Checks: serialized}, nil
}

func serializeChecks(checks []models.FactoryWorkOrderCheck) ([]*pb.WorkOrderCheck, error) {
	result := make([]*pb.WorkOrderCheck, 0, len(checks))
	for i := range checks {
		serialized, err := serializeCheck(&checks[i])
		if err != nil {
			return nil, err
		}
		result = append(result, serialized)
	}
	return result, nil
}

func serializeCheck(check *models.FactoryWorkOrderCheck) (*pb.WorkOrderCheck, error) {
	automation, err := check.AutomationRef()
	if err != nil {
		return nil, err
	}

	serialized := &pb.WorkOrderCheck{
		Id:            check.ID.String(),
		Key:           check.Key,
		Name:          check.Name,
		Score:         check.Score,
		MaxScore:      check.MaxScore,
		Format:        checkFormatToProto(check.Format),
		Level:         checkLevelToProto(check.Level),
		PreviousScore: check.PreviousScore,
		RecentScores:  check.RecentScores,
		Summary:       check.Summary,
		Analysis:      check.Analysis,
		CreatedAt:     timestamppb.New(check.CreatedAt),
		UpdatedAt:     timestamppb.New(check.UpdatedAt),
		Automation:    serializeAutomationRef(automation),
	}

	if check.RunID != nil {
		serialized.RunId = check.RunID.String()
	}

	return serialized, nil
}

func checkFormatToProto(format string) pb.WorkOrderCheck_Format {
	switch format {
	case models.FactoryWorkOrderCheckFormatFraction:
		return pb.WorkOrderCheck_FORMAT_FRACTION
	case models.FactoryWorkOrderCheckFormatPercent:
		return pb.WorkOrderCheck_FORMAT_PERCENT
	case models.FactoryWorkOrderCheckFormatBoolean:
		return pb.WorkOrderCheck_FORMAT_BOOLEAN
	default:
		return pb.WorkOrderCheck_FORMAT_UNSPECIFIED
	}
}

func checkLevelToProto(level string) pb.WorkOrderCheck_Level {
	switch level {
	case models.FactoryWorkOrderCheckLevelPositive:
		return pb.WorkOrderCheck_LEVEL_POSITIVE
	case models.FactoryWorkOrderCheckLevelNeutral:
		return pb.WorkOrderCheck_LEVEL_NEUTRAL
	case models.FactoryWorkOrderCheckLevelCaution:
		return pb.WorkOrderCheck_LEVEL_CAUTION
	case models.FactoryWorkOrderCheckLevelCritical:
		return pb.WorkOrderCheck_LEVEL_CRITICAL
	default:
		return pb.WorkOrderCheck_LEVEL_UNSPECIFIED
	}
}
