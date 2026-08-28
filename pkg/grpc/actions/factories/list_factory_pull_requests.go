package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactoryPullRequests(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryPullRequestsRequest,
) (*pb.ListFactoryPullRequestsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory pull requests")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory pull requests")
	}

	filter, err := listFactoryPullRequestFilter(req)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory pull requests")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory pull requests")
	}

	pullRequests, err := factory.ListPullRequests(db, filter)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory pull requests")
	}

	serialized, err := serializeFactoryPullRequests(db, pullRequests)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory pull requests")
	}

	return &pb.ListFactoryPullRequestsResponse{PullRequests: serialized}, nil
}

func listFactoryPullRequestFilter(req *pb.ListFactoryPullRequestsRequest) (models.FactoryPullRequestFilter, error) {
	hasOrder := req.Order != nil
	hasWorkOrderIDs := len(req.GetWorkOrderIds()) > 0
	if hasOrder && hasWorkOrderIDs {
		return models.FactoryPullRequestFilter{}, invalidArgument("order and work_order_ids cannot be set together")
	}

	filter := models.FactoryPullRequestFilter{}
	if hasOrder {
		orderNumber := req.GetOrder()
		if orderNumber <= 0 {
			return models.FactoryPullRequestFilter{}, invalidArgument("order must be a positive work order number")
		}
		filter.WorkOrderNumber = &orderNumber
	}
	if hasWorkOrderIDs {
		ids := make([]uuid.UUID, 0, len(req.GetWorkOrderIds()))
		for _, raw := range req.GetWorkOrderIds() {
			id, err := parseOrderID(raw)
			if err != nil {
				return models.FactoryPullRequestFilter{}, err
			}
			ids = append(ids, id)
		}
		filter.WorkOrderIDs = ids
	}
	return filter, nil
}
