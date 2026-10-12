package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

const maxWorkOrderListLimit = models.DefaultFactoryWorkOrderListLimit

func ListWorkOrders(ctx context.Context, organizationID string, req *pb.ListWorkOrdersRequest) (*pb.ListWorkOrdersResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	if req.GetBeforeId() != "" {
		if _, err := uuid.Parse(req.GetBeforeId()); err != nil {
			return nil, factoryErrorToStatus(invalidArgument("invalid before id"), "failed to list work orders")
		}
	}

	limit := workOrderListLimit(req.GetLimit())
	filters := listWorkOrderFilters(req)
	filters.Limit = limit + 1

	orders, err := factory.ListWorkOrders(db, filters)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	hasNextPage := false
	if len(orders) > limit {
		hasNextPage = true
		orders = orders[:limit]
	}

	var totalCount int64
	if req.GetBeforeId() == "" && !hasNextPage {
		totalCount = int64(len(orders))
	} else {
		totalCount, err = factory.CountWorkOrders(db, filters)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to list work orders")
		}
	}

	serialized, err := loadAndSerializeWorkOrders(ctx, factory, orders)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	return &pb.ListWorkOrdersResponse{
		Orders:      serialized,
		HasNextPage: hasNextPage,
		TotalCount:  uint32(totalCount),
	}, nil
}

func workOrderListLimit(limit uint32) int {
	if limit == 0 || limit > maxWorkOrderListLimit {
		return maxWorkOrderListLimit
	}
	return int(limit)
}
