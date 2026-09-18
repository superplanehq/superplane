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

	result, err := evaluateFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
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

	_, _, err = result.Client.MergePullRequest(ctx, pullRequest.Repository, int(pullRequest.Number), "", &github.PullRequestOptions{
		MergeMethod: method,
		SHA:         expectedSHA,
	})
	if err != nil {
		if isGitHubHeadMovedError(err) {
			return nil, factoryErrorToStatus(errFactoryPullRequestHeadMoved, "failed to merge factory pull request")
		}
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}

	state := models.FactoryPullRequestStateMerged
	mergedAt := time.Now()
	if err := pullRequest.Update(db, models.FactoryPullRequestPatch{
		State:    &state,
		MergedAt: &mergedAt,
	}); err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
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

func isGitHubHeadMovedError(err error) bool {
	var githubErr *github.ErrorResponse
	if errors.As(err, &githubErr) && githubErr.Response != nil {
		return githubErr.Response.StatusCode == http.StatusConflict
	}
	return false
}
