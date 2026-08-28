package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeFactoryPullRequest(
	ctx context.Context,
	organizationID string,
	req *pb.DescribeFactoryPullRequestRequest,
) (*pb.DescribeFactoryPullRequestResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request")
	}

	prID, err := parsePullRequestID(req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request")
	}

	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: prID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request")
	}

	serialized, err := serializeFactoryPullRequests(db, []models.FactoryPullRequest{*pullRequest})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request")
	}
	if len(serialized) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryPullRequestNotFound, "failed to describe factory pull request")
	}

	return &pb.DescribeFactoryPullRequestResponse{PullRequest: serialized[0]}, nil
}
