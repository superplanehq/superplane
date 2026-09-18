package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
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
	factory, pullRequest, err := loadFactoryPullRequestForMerge(db, orgID, req.GetFactoryId(), req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}

	result, err := evaluateFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}

	return &pb.DescribeFactoryPullRequestMergeabilityResponse{Mergeability: result.proto()}, nil
}
