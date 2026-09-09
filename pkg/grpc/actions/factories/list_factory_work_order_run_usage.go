package factories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	workOrderRunUsageDefaultDays     = 30
	workOrderRunUsagePageSizeDefault = 50
	workOrderRunUsagePageSizeMax     = 100
)

func ListFactoryWorkOrderRunUsage(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryWorkOrderRunUsageRequest,
) (*pb.ListFactoryWorkOrderRunUsageResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory work order run usage")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory work order run usage")
	}

	since, until, err := resolveWorkOrderRunUsageWindow(req)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory work order run usage")
	}

	rows, total, err := models.ListWorkOrderRunUsage(db, models.UsageReportFilter{
		OrganizationID: orgID,
		FactoryID:      &factoryID,
		Since:          since,
		Until:          until,
	}, clampWorkOrderRunUsagePageSize(int(req.GetLimit())), int(req.GetOffset()))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list factory work order run usage")
	}

	return &pb.ListFactoryWorkOrderRunUsageResponse{
		Rows:       serializeWorkOrderRunUsageRows(factory, rows),
		TotalCount: uint32(total),
	}, nil
}

func resolveWorkOrderRunUsageWindow(req *pb.ListFactoryWorkOrderRunUsageRequest) (time.Time, time.Time, error) {
	now := time.Now()
	until := now
	if req.GetEndTime() != nil {
		until = req.GetEndTime().AsTime()
	}

	since := until.AddDate(0, 0, -workOrderRunUsageDefaultDays)
	if req.GetStartTime() != nil {
		since = req.GetStartTime().AsTime()
	}

	if err := models.ValidateSpendingReportWindow(since, until); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return since, until, nil
}

func clampWorkOrderRunUsagePageSize(limit int) int {
	if limit <= 0 {
		return workOrderRunUsagePageSizeDefault
	}
	if limit > workOrderRunUsagePageSizeMax {
		return workOrderRunUsagePageSizeMax
	}
	return limit
}

func serializeWorkOrderRunUsageRows(factory *models.Factory, rows []models.WorkOrderRunUsage) []*pb.WorkOrderRunUsageRow {
	out := make([]*pb.WorkOrderRunUsageRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, serializeWorkOrderRunUsageRow(factory, row))
	}
	return out
}

func serializeWorkOrderRunUsageRow(factory *models.Factory, row models.WorkOrderRunUsage) *pb.WorkOrderRunUsageRow {
	item := &pb.WorkOrderRunUsageRow{
		WorkOrderExecutionId: row.WorkOrderExecutionID.String(),
		WorkOrderId:          row.WorkOrderID.String(),
		WorkOrderNumber:      row.WorkOrderNumber,
		WorkOrderKey:         factory.WorkOrderKey(row.WorkOrderNumber),
		Title:                row.Title,
		LastOccurredAt:       timestamppb.New(row.LastOccurredAt),
		UserName:             row.UserName,
		UserEmail:            row.UserEmail,
		TotalTokens:          row.TotalTokens,
		DurationSeconds:      row.DurationSeconds,
		CostCents:            row.CostCents(),
		HostedCostCents:      row.HostedCostCents(),
		ByokCostCents:        row.BYOKCostCents(),
		UsedByok:             row.UsedBYOK,
		Models:               row.Models,
		MachineTypes:         row.MachineTypes,
	}
	if row.UserID != nil && *row.UserID != uuid.Nil {
		item.UserId = row.UserID.String()
	}
	return item
}
