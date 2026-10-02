package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func RetryFactoryPullRequestWebhook(
	ctx context.Context,
	organizationID string,
	req *pb.RetryFactoryPullRequestWebhookRequest,
) (*pb.RetryFactoryPullRequestWebhookResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry factory pull request webhook")
	}

	db := database.DB(ctx)
	factory, pullRequest, err := loadFactoryPullRequestForMerge(db, orgID, req.GetFactoryId(), req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry factory pull request webhook")
	}

	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry factory pull request webhook")
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryPullRequestMerge) {
		return nil, factoryErrorToStatus(errFactoryPullRequestMergeDisabled, "failed to retry factory pull request webhook")
	}

	hook, err := findFactoryMergeabilityWebhookForPullRequest(db, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry factory pull request webhook")
	}
	if hook == nil || hook.State != models.WebhookStateFailed {
		return &pb.RetryFactoryPullRequestWebhookResponse{}, nil
	}
	if err := hook.ResetPending(db); err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry factory pull request webhook")
	}
	return &pb.RetryFactoryPullRequestWebhookResponse{}, nil
}
