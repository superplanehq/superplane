package factories

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

var (
	errFactoryPullRequestNotGitHub             = errors.New("only GitHub pull requests can merge from SuperPlane")
	errFactoryPullRequestNotOpen               = errors.New("the pull request is not open")
	errFactoryPullRequestNotMergeable          = errors.New("the pull request cannot merge")
	errFactoryPullRequestMergeMethodNotAllowed = errors.New("the repository does not allow this merge method")
	errFactoryPullRequestHeadMoved             = errors.New("the pull request head changed")
)

const (
	mergeBlockedActiveRun          = "Automation is still running."
	mergeBlockedChecksUnfinished   = "Checks are still running."
	mergeBlockedCheckFailed        = "A check failed."
	mergeBlockedDraft              = "The pull request is a draft."
	mergeBlockedConflicting        = "The pull request has conflicts."
	mergeBlockedMissingIntegration = "GitHub is not connected."
)

type factoryPullRequestMergeability struct {
	CanMerge       bool
	BlockedReason  pb.FactoryPullRequestMergeability_BlockedReason
	Message        string
	AllowedMethods []pb.FactoryPullRequestMergeability_MergeMethod
	HeadSHA        string
	PullRequest    *models.FactoryPullRequest
	Client         factoryGitHubAPI
}

func loadFactoryPullRequestForMerge(
	db *gorm.DB,
	orgID uuid.UUID,
	factoryID string,
	prID string,
) (*models.Factory, *models.FactoryPullRequest, error) {
	parsedPRID, err := parsePullRequestID(prID)
	if err != nil {
		return nil, nil, err
	}

	factory, err := findFactory(db, orgID, factoryID)
	if err != nil {
		return nil, nil, err
	}

	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parsedPRID})
	if err != nil {
		return nil, nil, err
	}
	if pullRequest.Provider != models.FactoryPullRequestProviderGitHub {
		return nil, nil, errFactoryPullRequestNotGitHub
	}
	if pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil, nil, errFactoryPullRequestNotOpen
	}
	return factory, pullRequest, nil
}

func evaluateFactoryPullRequestMergeability(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) (*factoryPullRequestMergeability, error) {
	result := &factoryPullRequestMergeability{PullRequest: pullRequest}

	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		if errors.Is(err, errFactoryGitHubNotConnected) {
			return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION, mergeBlockedMissingIntegration), nil
		}
		return nil, err
	}
	result.Client = client

	active, err := workOrderHasActiveRun(db, factory, pullRequest.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if active {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN, mergeBlockedActiveRun), nil
	}

	githubPR, _, err := client.GetPullRequest(ctx, pullRequest.Repository, int(pullRequest.Number))
	if err != nil {
		return nil, err
	}
	result.HeadSHA = githubPR.GetHead().GetSHA()

	if githubPR.GetDraft() || strings.EqualFold(githubPR.GetMergeableState(), "draft") {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_DRAFT, mergeBlockedDraft), nil
	}
	if isConflictingPullRequest(githubPR) {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CONFLICTING, mergeBlockedConflicting), nil
	}
	if strings.EqualFold(githubPR.GetMergeableState(), "unknown") || githubPR.Mergeable == nil {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, mergeBlockedChecksUnfinished), nil
	}

	unfinished, failed, err := evaluatePullRequestChecks(ctx, client, pullRequest.Repository, result.HeadSHA)
	if err != nil {
		return nil, err
	}
	if unfinished {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, mergeBlockedChecksUnfinished), nil
	}
	if failed {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED, mergeBlockedCheckFailed), nil
	}

	repository, err := client.FindRepository(pullRequest.Repository)
	if err != nil {
		return nil, err
	}
	result.AllowedMethods = allowedMergeMethods(repository)
	if len(result.AllowedMethods) == 0 {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNSPECIFIED, mergeBlockedConflicting), nil
	}

	result.CanMerge = true
	return result, nil
}

func blockedMergeability(
	result *factoryPullRequestMergeability,
	reason pb.FactoryPullRequestMergeability_BlockedReason,
	message string,
) *factoryPullRequestMergeability {
	result.CanMerge = false
	result.BlockedReason = reason
	result.Message = message
	return result
}

func workOrderHasActiveRun(db *gorm.DB, factory *models.Factory, workOrderID uuid.UUID) (bool, error) {
	order, err := factory.FindWorkOrder(db, workOrderID)
	if err != nil {
		return false, err
	}
	_, err = order.FindActiveLineDispatch(db)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func isConflictingPullRequest(pullRequest *github.PullRequest) bool {
	if pullRequest.Mergeable != nil && !pullRequest.GetMergeable() {
		return true
	}
	return strings.EqualFold(pullRequest.GetMergeableState(), "dirty")
}

func evaluatePullRequestChecks(
	ctx context.Context,
	client factoryGitHubAPI,
	repository string,
	sha string,
) (unfinished bool, failed bool, err error) {
	if sha == "" {
		return true, false, nil
	}

	combined, _, err := client.GetCombinedStatus(ctx, repository, sha, &github.ListOptions{PerPage: 100})
	if err != nil {
		return false, false, err
	}
	unfinished, failed = combinedStatusGate(combined)

	checks, _, err := client.ListCheckRunsForRef(ctx, repository, sha, &github.ListCheckRunsOptions{
		Filter:      github.Ptr("latest"),
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return false, false, err
	}
	checkUnfinished, checkFailed := checkRunGate(checks)
	return unfinished || checkUnfinished, failed || checkFailed, nil
}

func combinedStatusGate(combined *github.CombinedStatus) (unfinished bool, failed bool) {
	if combined == nil {
		return false, false
	}
	switch strings.ToLower(combined.GetState()) {
	case "pending":
		unfinished = true
	case "failure", "error":
		failed = true
	}
	for _, status := range combined.Statuses {
		if status == nil {
			continue
		}
		switch strings.ToLower(status.GetState()) {
		case "pending":
			unfinished = true
		case "failure", "error":
			failed = true
		}
	}
	return unfinished, failed
}

func checkRunGate(results *github.ListCheckRunsResults) (unfinished bool, failed bool) {
	if results == nil {
		return false, false
	}
	for _, run := range results.CheckRuns {
		if run == nil {
			continue
		}
		switch strings.ToLower(run.GetStatus()) {
		case "queued", "in_progress", "waiting", "requested", "pending":
			unfinished = true
		}
		if !strings.EqualFold(run.GetStatus(), "completed") {
			continue
		}
		switch strings.ToLower(run.GetConclusion()) {
		case "failure", "timed_out", "cancelled", "canceled":
			failed = true
		}
	}
	return unfinished, failed
}

func allowedMergeMethods(repository *github.Repository) []pb.FactoryPullRequestMergeability_MergeMethod {
	if repository == nil {
		return nil
	}
	methods := make([]pb.FactoryPullRequestMergeability_MergeMethod, 0, 3)
	if repository.AllowSquashMerge == nil || repository.GetAllowSquashMerge() {
		methods = append(methods, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH)
	}
	if repository.AllowMergeCommit == nil || repository.GetAllowMergeCommit() {
		methods = append(methods, pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE)
	}
	if repository.AllowRebaseMerge == nil || repository.GetAllowRebaseMerge() {
		methods = append(methods, pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE)
	}
	return methods
}

func mergeMethodToGitHub(method pb.FactoryPullRequestMergeability_MergeMethod) (string, bool) {
	switch method {
	case pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH:
		return "squash", true
	case pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE:
		return "merge", true
	case pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE:
		return "rebase", true
	default:
		return "", false
	}
}

func mergeMethodAllowed(methods []pb.FactoryPullRequestMergeability_MergeMethod, method pb.FactoryPullRequestMergeability_MergeMethod) bool {
	return slices.Contains(methods, method)
}

func (m *factoryPullRequestMergeability) proto() *pb.FactoryPullRequestMergeability {
	if m == nil {
		return &pb.FactoryPullRequestMergeability{}
	}
	return &pb.FactoryPullRequestMergeability{
		CanMerge:       m.CanMerge,
		BlockedReason:  m.BlockedReason,
		Message:        m.Message,
		AllowedMethods: m.AllowedMethods,
		HeadSha:        m.HeadSHA,
	}
}
