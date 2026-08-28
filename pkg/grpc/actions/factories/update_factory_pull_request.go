package factories

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func UpdateFactoryPullRequest(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateFactoryPullRequestRequest,
) (*pb.UpdateFactoryPullRequestResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}

	prID, err := parsePullRequestID(req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}

	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: prID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}

	patch := models.FactoryPullRequestPatch{
		ExternalID: req.ExternalId,
		Repository: req.Repository,
		URL:        req.Url,
		Title:      req.Title,
		MergedAt:   timestampPointer(req.GetMergedAt()),
		ClosedAt:   timestampPointer(req.GetClosedAt()),
	}
	if req.State != nil {
		state := pullRequestStateFromProto(req.GetState())
		if state == "" {
			return nil, factoryErrorToStatus(invalidArgument("invalid pull request state"), "failed to update factory pull request")
		}
		patch.State = &state
	}

	if err := pullRequest.Update(db, patch); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		pullRequest.WorkOrderID.String(),
		factoryevents.EventTypeOrderPullRequestUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", pullRequest.WorkOrderID)
	}

	serialized, err := serializeFactoryPullRequests(db, []models.FactoryPullRequest{*pullRequest})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory pull request")
	}
	if len(serialized) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryPullRequestNotFound, "failed to update factory pull request")
	}

	return &pb.UpdateFactoryPullRequestResponse{PullRequest: serialized[0]}, nil
}
