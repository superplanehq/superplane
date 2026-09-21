package factories

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func MergeFactoryPullRequest(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.MergeFactoryPullRequestRequest,
) (*pb.MergeFactoryPullRequestResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}

	db := database.DB(ctx)
	factory, pullRequest, err := loadFactoryPullRequestForMerge(db, orgID, req.GetFactoryId(), req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}

	result, cached, err := mergeabilityFromCache(db, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	if !cached {
		result, err = syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
		}
	}
	if !result.CanMerge {
		message := result.Message
		if message == "" {
			message = errFactoryPullRequestNotMergeable.Error()
		}
		return nil, factoryErrorToStatus(errors.Join(errFactoryPullRequestNotMergeable, errors.New(message)), "failed to merge factory pull request")
	}

	method, ok := mergeMethodToGitHub(req.GetMergeMethod())
	if !ok {
		return nil, factoryErrorToStatus(invalidArgument("invalid merge method"), "failed to merge factory pull request")
	}
	if !mergeMethodAllowed(result.AllowedMethods, req.GetMergeMethod()) {
		return nil, factoryErrorToStatus(errFactoryPullRequestMergeMethodNotAllowed, "failed to merge factory pull request")
	}

	expectedSHA := req.GetExpectedHeadSha()
	if expectedSHA == "" {
		return nil, factoryErrorToStatus(invalidArgument("expected head SHA is required"), "failed to merge factory pull request")
	}
	if result.HeadSHA != expectedSHA {
		return nil, factoryErrorToStatus(errFactoryPullRequestHeadMoved, "failed to merge factory pull request")
	}

	if result.Client == nil {
		client, err := newFactoryGitHubAPI(db, deps, factory)
		if err != nil {
			if errors.Is(err, errFactoryGitHubNotConnected) {
				return nil, factoryErrorToStatus(
					errors.Join(errFactoryPullRequestNotMergeable, errors.New(mergeBlockedMissingIntegration)),
					"failed to merge factory pull request",
				)
			}
			return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
		}
		result.Client = client
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := assertNoActiveAutomationLocked(tx, factory, pullRequest); err != nil {
			return err
		}

		_, _, err := result.Client.MergePullRequest(ctx, pullRequest.Repository, int(pullRequest.Number), "", &github.PullRequestOptions{
			MergeMethod: method,
			SHA:         expectedSHA,
		})
		if err != nil {
			return err
		}

		state := models.FactoryPullRequestStateMerged
		mergedAt := time.Now()
		return pullRequest.Update(tx, models.FactoryPullRequestPatch{
			State:    &state,
			MergedAt: &mergedAt,
		})
	})
	if err != nil {
		if errors.Is(err, errFactoryPullRequestNotMergeable) {
			return nil, factoryErrorToStatus(errors.Join(errFactoryPullRequestNotMergeable, errors.New(mergeBlockedActiveRun)), "failed to merge factory pull request")
		}
		if isGitHubHeadMovedError(err) {
			_, _ = syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
			return nil, factoryErrorToStatus(errFactoryPullRequestHeadMoved, "failed to merge factory pull request")
		}
		message := errFactoryPullRequestNotMergeable.Error()
		if synced, syncErr := syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest); syncErr == nil {
			if !synced.CanMerge && synced.Message != "" {
				message = synced.Message
			} else if synced.CanMerge {
				_ = pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
					BlockedMessage: message,
					HeadSHA:        pullRequest.MergeableHeadSHA,
				})
			}
		}
		return nil, factoryErrorToStatus(errors.Join(errFactoryPullRequestNotMergeable, errors.New(message)), "failed to merge factory pull request")
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
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	if len(serialized) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryPullRequestNotFound, "failed to merge factory pull request")
	}

	return &pb.MergeFactoryPullRequestResponse{PullRequest: serialized[0]}, nil
}

func assertNoActiveAutomationLocked(tx *gorm.DB, factory *models.Factory, pullRequest *models.FactoryPullRequest) error {
	order, err := factory.FindWorkOrder(tx, pullRequest.WorkOrderID)
	if err != nil {
		return err
	}
	if err := order.LockForUpdate(tx); err != nil {
		return err
	}
	if err := pullRequest.LockForUpdate(tx); err != nil {
		return err
	}
	active, err := factoryPullRequestHasActiveAutomation(tx, factory, pullRequest)
	if err != nil {
		return err
	}
	if active {
		return errFactoryPullRequestNotMergeable
	}
	return nil
}

func isGitHubHeadMovedError(err error) bool {
	var githubErr *github.ErrorResponse
	if errors.As(err, &githubErr) && githubErr.Response != nil {
		return githubErr.Response.StatusCode == http.StatusConflict
	}
	return false
}
