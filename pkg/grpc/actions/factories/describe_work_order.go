package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeWorkOrder(ctx context.Context, organizationID string, req *pb.DescribeWorkOrderRequest) (*pb.DescribeWorkOrderResponse, error) {
	loaded, err := loadDescribedWorkOrder(ctx, organizationID, req, false)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeWorkOrderResponse{Order: loaded.Order}, nil
}

type loadedWorkOrder struct {
	Factory *models.Factory
	Record  *models.FactoryWorkOrder
	Order   *pb.WorkOrder
}

func loadDescribedWorkOrder(ctx context.Context, organizationID string, req *pb.DescribeWorkOrderRequest, includeArchived bool) (*loadedWorkOrder, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	order, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	if !includeArchived && order.IsArchived() {
		return nil, grpcerrors.NotFound(models.ErrFactoryWorkOrderNotFound, "work order not found")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	return &loadedWorkOrder{Factory: factory, Record: order, Order: serialized}, nil
}
