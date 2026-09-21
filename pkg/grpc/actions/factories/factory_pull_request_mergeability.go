package factories

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

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
	mergeBlockedNotOpen            = "The pull request is not open."
)

type factoryPullRequestMergeability struct {
	CanMerge       bool
	BlockedReason  pb.FactoryPullRequestMergeability_BlockedReason
	Message        string
	AllowedMethods []pb.FactoryPullRequestMergeability_MergeMethod
	HeadSHA        string
	PullRequest    *models.FactoryPullRequest
	Client         factoryGitHubAPI
	StateCorrected bool
}

func loadFactoryPullRequestForMerge(
	db *gorm.DB,
	orgID uuid.UUID,
	factoryID string,
	prID string,
) (*models.Factory, *models.FactoryPullRequest, error) {
	factory, pullRequest, err := loadFactoryPullRequest(db, orgID, factoryID, prID)
	if err != nil {
		return nil, nil, err
	}
	if pullRequest.State != models.FactoryPullRequestStateOpen {
		return nil, nil, errFactoryPullRequestNotOpen
	}
	return factory, pullRequest, nil
}

func loadFactoryPullRequestForDescribe(
	db *gorm.DB,
	orgID uuid.UUID,
	factoryID string,
	prID string,
) (*models.Factory, *models.FactoryPullRequest, error) {
	return loadFactoryPullRequest(db, orgID, factoryID, prID)
}

func loadFactoryPullRequest(
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
	if pullRequest.State != models.FactoryPullRequestStateOpen {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_NOT_OPEN, mergeBlockedNotOpen), nil
	}

	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		if errors.Is(err, errFactoryGitHubNotConnected) {
			return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION, mergeBlockedMissingIntegration), nil
		}
		return nil, err
	}
	result.Client = client

	active, err := factoryPullRequestHasActiveAutomation(db, factory, pullRequest)
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

	if githubPullRequestIsClosed(githubPR) {
		corrected, err := syncFactoryPullRequestIfClosedOnGitHub(db, pullRequest, githubPR)
		if err != nil {
			return nil, err
		}
		result.StateCorrected = corrected
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_NOT_OPEN, mergeBlockedNotOpen), nil
	}

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

func githubPullRequestIsClosed(githubPR *github.PullRequest) bool {
	_, ok := factoryPullRequestStateFromGitHub(githubPR)
	return ok
}

func factoryPullRequestStateFromGitHub(githubPR *github.PullRequest) (string, bool) {
	if githubPR.GetMerged() {
		return models.FactoryPullRequestStateMerged, true
	}
	if strings.EqualFold(githubPR.GetState(), "closed") {
		return models.FactoryPullRequestStateClosed, true
	}
	return "", false
}

func syncFactoryPullRequestIfClosedOnGitHub(
	db *gorm.DB,
	pullRequest *models.FactoryPullRequest,
	githubPR *github.PullRequest,
) (bool, error) {
	nextState, ok := factoryPullRequestStateFromGitHub(githubPR)
	if !ok || pullRequest.State == nextState {
		return false, nil
	}

	var corrected bool
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := pullRequest.LockForUpdate(tx); err != nil {
			return err
		}
		if pullRequest.State != models.FactoryPullRequestStateOpen {
			return nil
		}

		patch := models.FactoryPullRequestPatch{State: &nextState}
		if nextState == models.FactoryPullRequestStateMerged {
			patch.MergedAt = githubTimePointer(githubPR.GetMergedAt())
		}
		if nextState == models.FactoryPullRequestStateClosed {
			patch.ClosedAt = githubTimePointer(githubPR.GetClosedAt())
		}
		if err := pullRequest.Update(tx, patch); err != nil {
			return err
		}
		corrected = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return corrected, nil
}

func githubTimePointer(ts github.Timestamp) *time.Time {
	if ts.Time.IsZero() {
		return nil
	}
	stamp := ts.Time
	return &stamp
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

func factoryPullRequestHasActiveAutomation(db *gorm.DB, factory *models.Factory, pullRequest *models.FactoryPullRequest) (bool, error) {
	if pullRequest.ActiveMutationRunID != nil {
		return true, nil
	}
	return workOrderHasActiveRun(db, factory, pullRequest.WorkOrderID)
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

	unfinished, failed, err = evaluateCombinedStatuses(ctx, client, repository, sha)
	if err != nil {
		return false, false, err
	}

	checkUnfinished, checkFailed, err := evaluateCheckRuns(ctx, client, repository, sha)
	if err != nil {
		return false, false, err
	}
	return unfinished || checkUnfinished, failed || checkFailed, nil
}

func evaluateCombinedStatuses(
	ctx context.Context,
	client factoryGitHubAPI,
	repository string,
	sha string,
) (unfinished bool, failed bool, err error) {
	opts := &github.ListOptions{PerPage: 100}
	for {
		combined, response, err := client.GetCombinedStatus(ctx, repository, sha, opts)
		if err != nil {
			return false, false, err
		}
		pageUnfinished, pageFailed := combinedStatusGate(combined)
		unfinished = unfinished || pageUnfinished
		failed = failed || pageFailed
		if response == nil || response.NextPage == 0 {
			return unfinished, failed, nil
		}
		opts.Page = response.NextPage
	}
}

func evaluateCheckRuns(
	ctx context.Context,
	client factoryGitHubAPI,
	repository string,
	sha string,
) (unfinished bool, failed bool, err error) {
	opts := &github.ListCheckRunsOptions{
		Filter:      github.Ptr("latest"),
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		checks, response, err := client.ListCheckRunsForRef(ctx, repository, sha, opts)
		if err != nil {
			return false, false, err
		}
		pageUnfinished, pageFailed := checkRunGate(checks)
		unfinished = unfinished || pageUnfinished
		failed = failed || pageFailed
		if response == nil || response.NextPage == 0 {
			return unfinished, failed, nil
		}
		opts.Page = response.NextPage
	}
}

func combinedStatusGate(combined *github.CombinedStatus) (unfinished bool, failed bool) {
	if combined == nil {
		return false, false
	}
	// GitHub sets combined state to "pending" when the commit has no statuses.
	// That rollup is not an unfinished check.
	if len(combined.Statuses) == 0 {
		return false, false
	}
	stateUnfinished, stateFailed := classifyCommitStatus(combined.GetState())
	unfinished = stateUnfinished
	failed = stateFailed
	for _, status := range combined.Statuses {
		if status == nil {
			continue
		}
		statusUnfinished, statusFailed := classifyCommitStatus(status.GetState())
		unfinished = unfinished || statusUnfinished
		failed = failed || statusFailed
	}
	return unfinished, failed
}

func classifyCommitStatus(state string) (unfinished bool, failed bool) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "success":
		return false, false
	case "pending":
		return true, false
	case "failure", "error":
		return false, true
	default:
		return true, false
	}
}

func checkRunGate(results *github.ListCheckRunsResults) (unfinished bool, failed bool) {
	if results == nil {
		return false, false
	}
	for _, run := range results.CheckRuns {
		if run == nil {
			continue
		}
		runUnfinished, runFailed := classifyCheckRun(run)
		unfinished = unfinished || runUnfinished
		failed = failed || runFailed
	}
	return unfinished, failed
}

func classifyCheckRun(run *github.CheckRun) (unfinished bool, failed bool) {
	if !strings.EqualFold(run.GetStatus(), "completed") {
		return true, false
	}
	switch strings.ToLower(strings.TrimSpace(run.GetConclusion())) {
	case "success", "neutral", "skipped":
		return false, false
	case "failure", "error", "timed_out", "action_required", "cancelled", "canceled":
		return false, true
	default:
		return true, false
	}
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
