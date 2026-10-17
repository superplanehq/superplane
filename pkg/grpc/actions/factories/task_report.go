package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

// TaskReport is the recorded task, its usage, and one page of associated runs.
type TaskReport struct {
	Factory       *models.Factory
	Order         *pb.WorkOrder
	UsageRecorded bool
	Runs          models.TaskRunHistory
}

// BuildTaskReport loads one task, including an archived task, and its run history.
// Usage totals come from the task ledger. Run history is a separate page.
func BuildTaskReport(ctx context.Context, organizationID, factoryID, orderRef string, history models.TaskRunHistoryQuery) (*TaskReport, error) {
	loaded, err := loadDescribedWorkOrder(ctx, organizationID, &pb.DescribeWorkOrderRequest{
		FactoryId: factoryID,
		OrderId:   orderRef,
	}, true)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	recorded, err := models.WorkOrderHasUsageEvents(db, loaded.Record.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	runs, err := loaded.Record.ListTaskRunHistory(db, history)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	return &TaskReport{
		Factory:       loaded.Factory,
		Order:         loaded.Order,
		UsageRecorded: recorded,
		Runs:          runs,
	}, nil
}
