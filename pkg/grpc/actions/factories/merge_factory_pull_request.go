package factories

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
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

	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryPullRequestMerge) {
		return nil, factoryErrorToStatus(errFactoryPullRequestMergeDisabled, "failed to merge factory pull request")
	}

	provider := openFactoryVCS(db, deps, factory)
	// Bitbucket mergeability always reads fresh. Cached snapshots predate
	// merge support and would keep a mergeable pull request blocked.
	_, freshOnly := provider.(*bitbucketProvider)
	var result *factoryPullRequestMergeability
	var cached bool
	if !freshOnly {
		result, cached, err = mergeabilityFromCache(db, factory, pullRequest)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
		}
	}
	if !cached {
		result, err = readFactoryPullRequestMergeability(ctx, db, provider, pullRequest)
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

	if err := connectFactoryVCS(provider); err != nil {
		if errors.Is(err, errFactoryGitHubNotConnected) {
			return nil, factoryErrorToStatus(
				errors.Join(errFactoryPullRequestNotMergeable, errors.New(mergeBlockedMissingIntegration)),
				"failed to merge factory pull request",
			)
		}
		if errors.Is(err, errFactoryBitbucketNotConnected) {
			return nil, factoryErrorToStatus(
				errors.Join(errFactoryPullRequestNotMergeable, errors.New(bitbucketMergeBlockedMissingIntegration)),
				"failed to merge factory pull request",
			)
		}
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}

	// The provider call stays outside the organization lock. A slow Git host
	// must not block dispatch or cancellation for every other task. This task
	// keeps a separate lock until the host accepts or rejects the merge, so
	// its line and root events cannot start in that gap.
	release, err := lockWorkOrderForMerge(ctx, db, models.WorkOrderMergeAdvisoryKey(pullRequest.WorkOrderID))
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	defer release()
	if err := rejectMergeWhileAutomationActive(db, factory, pullRequest); err != nil {
		return nil, mergeFactoryPullRequestError(ctx, db, deps, factory, pullRequest, err)
	}
	if err := provider.MergePullRequest(ctx, pullRequest, method, expectedSHA); err != nil {
		return nil, mergeFactoryPullRequestError(ctx, db, deps, factory, pullRequest, err)
	}

	closeOutcome, err := recordMergedFactoryPullRequest(db, factory, pullRequest)
	if err != nil {
		return nil, mergeFactoryPullRequestError(ctx, db, deps, factory, pullRequest, err)
	}

	if closeOutcome != nil && closeOutcome.closed {
		publishWorkOrderClosed(
			factory.OrganizationID,
			factory,
			closeOutcome.order,
			nil,
			closeOutcome.fromState,
			closeOutcome.result,
			false,
		)
	}
	captureFactoryProductEvent(
		ctx,
		db,
		factory,
		pullRequest.WorkOrderID,
		productanalytics.EventFactoryTaskMerged,
		productanalytics.StableEventID(productanalytics.EventFactoryTaskMerged, pullRequest.ID.String()),
		productanalytics.Properties{
			"provider":     pullRequest.Provider,
			"merge_source": "superplane",
		},
	)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		pullRequest.WorkOrderID.String(),
		factoryevents.EventTypeOrderPullRequestUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", pullRequest.WorkOrderID)
	}

	serialized, err := serializeFactoryPullRequests(ctx, db, []models.FactoryPullRequest{*pullRequest}, nil)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	if len(serialized) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryPullRequestNotFound, "failed to merge factory pull request")
	}

	return &pb.MergeFactoryPullRequestResponse{PullRequest: serialized[0]}, nil
}

func rejectMergeWhileAutomationActive(db *gorm.DB, factory *models.Factory, pullRequest *models.FactoryPullRequest) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := models.LockOrganizationTaskAdmission(tx, factory.OrganizationID); err != nil {
			return err
		}
		if err := assertNoActiveAutomationLocked(tx, factory, pullRequest); err != nil {
			return err
		}
		if pullRequest.State != models.FactoryPullRequestStateOpen {
			return errFactoryPullRequestNotOpen
		}
		return nil
	})
}

func lockWorkOrderForMerge(ctx context.Context, db *gorm.DB, key int64) (func(), error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key)
		_ = conn.Close()
	}, nil
}

func recordMergedFactoryPullRequest(
	db *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) (*factoryPullRequestCloseResult, error) {
	var closeOutcome *factoryPullRequestCloseResult
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := models.LockOrganizationTaskAdmission(tx, factory.OrganizationID); err != nil {
			return err
		}
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
		if pullRequest.State != models.FactoryPullRequestStateMerged {
			state := models.FactoryPullRequestStateMerged
			mergedAt := time.Now()
			if err := pullRequest.Update(tx, models.FactoryPullRequestPatch{
				State:    &state,
				MergedAt: &mergedAt,
			}); err != nil {
				return err
			}
		}
		outcome, err := closeFactoryWorkOrderForPullRequest(tx, factory, pullRequest, true, nil)
		if err != nil {
			return err
		}
		closeOutcome = outcome
		return nil
	})
	return closeOutcome, err
}

func mergeFactoryPullRequestError(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	err error,
) error {
	if errors.Is(err, vcs.ErrNotSupported) {
		return factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	if errors.Is(err, errFactoryPullRequestNotMergeable) {
		return factoryErrorToStatus(errors.Join(errFactoryPullRequestNotMergeable, errors.New(mergeBlockedActiveRun)), "failed to merge factory pull request")
	}
	if errors.Is(err, errFactoryPullRequestHeadMoved) {
		return factoryErrorToStatus(err, "failed to merge factory pull request")
	}
	if isGitHubHeadMovedError(err) {
		_, _ = syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
		return factoryErrorToStatus(errFactoryPullRequestHeadMoved, "failed to merge factory pull request")
	}
	message := errFactoryPullRequestNotMergeable.Error()
	if synced, syncErr := syncFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest); syncErr == nil {
		if !synced.CanMerge && synced.Message != "" {
			message = synced.Message
		} else if synced.CanMerge {
			_ = pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
				BlockedMessage: message,
				HeadSHA:        pullRequest.MergeableHeadSHA,
				AllowedMethods: pullRequest.MergeableAllowedMethods,
			})
		}
	}
	return factoryErrorToStatus(errors.Join(errFactoryPullRequestNotMergeable, errors.New(message)), "failed to merge factory pull request")
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
