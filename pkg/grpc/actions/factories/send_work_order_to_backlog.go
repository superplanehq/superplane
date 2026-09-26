package factories

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func SendWorkOrderToBacklog(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.SendWorkOrderToBacklogRequest,
) (*pb.SendWorkOrderToBacklogResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actor, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to send work order to backlog")
	}

	db := database.DB(ctx)
	resolvedFactory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}
	factoryID := resolvedFactory.ID

	resolvedOrder, err := findWorkOrder(db, resolvedFactory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}
	orderID := resolvedOrder.ID
	if resolvedOrder.State != models.FactoryWorkOrderStateClosed {
		return nil, factoryErrorToStatus(errWorkOrderNotClosedForBacklog, "failed to send work order to backlog")
	}

	grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{orderID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}
	closeable := closeablePullRequests(grouped[orderID])

	if req.GetClosePullRequests() && len(closeable) > 0 {
		if err := closePreviousPullRequests(ctx, db, deps, resolvedFactory, closeable); err != nil {
			return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
		}
	}

	fromState := resolvedOrder.State
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}
		order, err := factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}
		if req.GetClosePullRequests() {
			now := time.Now()
			for i := range closeable {
				if err := stampFactoryPullRequestClosed(tx, &closeable[i], false, nil, &now); err != nil {
					return err
				}
			}
		}
		if req.GetClearArtifacts() {
			if _, err := order.DeleteArtifacts(tx, &actor); err != nil {
				return err
			}
		}
		_, err = order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateDraft,
			Actor:   &actor,
		})
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}

	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}
	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: orgID.String(),
		FactoryID:      factory.ID.String(),
		OrderID:        order.ID.String(),
		EventType:      factoryevents.EventTypeOrderStatusUpdated,
		ActorUserID:    actor.String(),
		FromState:      fromState,
		ToState:        models.FactoryWorkOrderStateDraft,
	}
	if err := notification.Publish(); err != nil {
		log.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to send work order to backlog")
	}

	return &pb.SendWorkOrderToBacklogResponse{Order: serialized}, nil
}

func closeablePullRequests(pullRequests []models.FactoryPullRequest) []models.FactoryPullRequest {
	closeable := make([]models.FactoryPullRequest, 0, len(pullRequests))
	for _, pullRequest := range pullRequests {
		if pullRequest.State == models.FactoryPullRequestStateOpen || pullRequest.State == models.FactoryPullRequestStateDraft {
			closeable = append(closeable, pullRequest)
		}
	}
	return closeable
}

func closePreviousPullRequests(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequests []models.FactoryPullRequest,
) error {
	for _, pullRequest := range pullRequests {
		if pullRequest.Provider == models.FactoryPullRequestProviderBitbucket {
			return errCannotCloseBitbucketPullRequest
		}
		if pullRequest.Provider != models.FactoryPullRequestProviderGitHub {
			return errCannotClosePullRequest
		}
	}

	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return err
	}

	closed := github.Ptr("closed")
	for _, pullRequest := range pullRequests {
		_, _, err := client.EditPullRequest(ctx, pullRequest.Repository, int(pullRequest.Number), &github.PullRequest{
			State: closed,
		})
		if err != nil {
			return errors.Join(errCannotClosePullRequest, err)
		}
	}
	return nil
}
