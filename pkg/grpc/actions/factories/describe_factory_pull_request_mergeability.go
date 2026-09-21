package factories

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeFactoryPullRequestMergeability(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.DescribeFactoryPullRequestMergeabilityRequest,
) (*pb.DescribeFactoryPullRequestMergeabilityResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}

	db := database.DB(ctx)
	factory, pullRequest, err := loadFactoryPullRequestForDescribe(db, orgID, req.GetFactoryId(), req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}

	result, err := evaluateFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}

	if result.StateCorrected {
		if err := messages.PublishFactoryWorkOrderUpdated(
			factory.ID.String(),
			pullRequest.WorkOrderID.String(),
			factoryevents.EventTypeOrderPullRequestUpdated,
		); err != nil {
			log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", pullRequest.WorkOrderID)
		}
	}

	return &pb.DescribeFactoryPullRequestMergeabilityResponse{Mergeability: result.proto()}, nil
}
