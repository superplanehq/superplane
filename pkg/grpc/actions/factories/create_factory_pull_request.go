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

func CreateFactoryPullRequest(
	ctx context.Context,
	organizationID string,
	req *pb.CreateFactoryPullRequestRequest,
) (*pb.CreateFactoryPullRequestResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}

	orderID, err := parseOrderID(req.GetWorkOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}

	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}

	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   pullRequestProviderFromProto(req.GetProvider()),
		ExternalID: req.GetExternalId(),
		Repository: req.GetRepository(),
		Number:     req.GetNumber(),
		URL:        req.GetUrl(),
		Title:      req.GetTitle(),
		State:      pullRequestStateFromProto(req.GetState()),
		MergedAt:   timestampPointer(req.GetMergedAt()),
		ClosedAt:   timestampPointer(req.GetClosedAt()),
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderPullRequestAdded,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	serialized, err := serializeFactoryPullRequests(db, []models.FactoryPullRequest{*pullRequest})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory pull request")
	}
	if len(serialized) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryPullRequestNotFound, "failed to create factory pull request")
	}

	return &pb.CreateFactoryPullRequestResponse{PullRequest: serialized[0]}, nil
}
