package factories

import (
	"context"
	"errors"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
)

func DescribeFactoryPullRequestMergeability(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.DescribeFactoryPullRequestMergeabilityRequest,
) (_ *pb.DescribeFactoryPullRequestMergeabilityResponse, err error) {
	repository := ""
	defer func() {
		if canceled, _ := grpcerrors.StatusFromContextError(ctx, err); canceled {
			return
		}
		if code, _, ok := grpcerrors.HandlerStatus(err); ok && code == codes.Internal {
			recordFactoryPullRequestMergeabilityError(ctx, req.GetFactoryId(), req.GetPrId(), repository, errors.Unwrap(err))
		}
	}()
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}

	db := database.DB(ctx)
	factory, pullRequest, err := loadFactoryPullRequestForMerge(db, orgID, req.GetFactoryId(), req.GetPrId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}
	repository = pullRequest.Repository

	failedWebhook, err := findFactoryMergeabilityWebhookForPullRequest(db, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}
	if failedWebhook != nil && failedWebhook.State == models.WebhookStateFailed {
		return &pb.DescribeFactoryPullRequestMergeabilityResponse{
			Mergeability: failedFactoryMergeabilityResult(pullRequest, failedWebhook).proto(),
		}, nil
	}

	result, cached, err := mergeabilityFromCache(db, factory, pullRequest)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
	}
	if !cached {
		result, err = syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to describe factory pull request mergeability")
		}
	}

	return &pb.DescribeFactoryPullRequestMergeabilityResponse{Mergeability: result.proto()}, nil
}

func recordFactoryPullRequestMergeabilityError(ctx context.Context, factoryID, pullRequestID, repository string, err error) {
	tags := map[string]string{
		"factory_id":      factoryID,
		"pull_request_id": pullRequestID,
		"repository":      repository,
	}
	log.WithError(err).WithFields(log.Fields{
		"factory_id":      factoryID,
		"pull_request_id": pullRequestID,
		"repository":      repository,
	}).Warn("factory pull request mergeability lookup failed")
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	if hub == nil || hub.Client() == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTags(tags)
		hub.CaptureException(err)
	})
}
