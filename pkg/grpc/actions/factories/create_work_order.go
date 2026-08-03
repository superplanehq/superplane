package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CreateWorkOrder(ctx context.Context, organizationID string, req *pb.CreateWorkOrderRequest) (*pb.CreateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, factoryErrorToStatus(invalidArgument("title is required"), "failed to create work order")
	}

	tx := database.DB(ctx)
	factory, err := loadFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	assigneeIDs, err := parseAssigneeIDs(tx, orgID, req.GetAssigneeIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order")
	}

	order, err := factory.CreateWorkOrder(tx, title, req.GetDescription(), createdByID, assigneeIDs)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	return &pb.CreateWorkOrderResponse{
		Order: serialized,
	}, nil
}
