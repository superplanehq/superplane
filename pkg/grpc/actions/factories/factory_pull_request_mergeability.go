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

func persistFactoryPullRequestMergeability(
	db *gorm.DB,
	pullRequest *models.FactoryPullRequest,
	result *factoryPullRequestMergeability,
) error {
	if pullRequest == nil || result == nil {
		return nil
	}
	if result.BlockedReason == pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN ||
		result.BlockedReason == pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION {
		return nil
	}
	return pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable:      result.CanMerge,
		BlockedReason:  mergeabilityBlockedReasonName(result.BlockedReason),
		BlockedMessage: result.Message,
		HeadSHA:        result.HeadSHA,
		AllowedMethods: mergeMethodNames(result.AllowedMethods),
	})
}

func syncFactoryPullRequestMergeability(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) (*factoryPullRequestMergeability, error) {
	result, err := evaluateFactoryPullRequestMergeability(ctx, db, deps, factory, pullRequest)
	if err != nil {
		return nil, err
	}
	if err := persistFactoryPullRequestMergeability(db, pullRequest, result); err != nil {
		return nil, err
	}
	return result, nil
}

func mergeabilityFromCache(
	db *gorm.DB,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
) (*factoryPullRequestMergeability, bool, error) {
	result := &factoryPullRequestMergeability{
		PullRequest:    pullRequest,
		HeadSHA:        pullRequest.MergeableHeadSHA,
		AllowedMethods: mergeMethodsFromNames(pullRequest.CachedAllowedMethods()),
	}
	active, err := factoryPullRequestHasActiveAutomation(db, factory, pullRequest)
	if err != nil {
		return nil, false, err
	}
	if active {
		return blockedMergeability(result, pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN, mergeBlockedActiveRun), true, nil
	}
	if !pullRequest.HasCachedMergeability() {
		return result, false, nil
	}
	matchesHead, err := cachedMergeabilityMatchesHead(db, pullRequest)
	if err != nil {
		return nil, false, err
	}
	if !matchesHead {
		return result, false, nil
	}
	reason := mergeabilityBlockedReasonFromName(pullRequest.MergeBlockedReason)
	if reason == pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN ||
		reason == pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION {
		return result, false, nil
	}
	if pullRequest.Mergeable {
		if len(result.AllowedMethods) == 0 {
			return result, false, nil
		}
		result.CanMerge = true
		return result, true, nil
	}
	return blockedMergeability(result, reason, pullRequest.MergeBlockedMessage), true, nil
}

func cachedMergeabilityMatchesHead(db *gorm.DB, pullRequest *models.FactoryPullRequest) (bool, error) {
	if pullRequest.CurrentRevisionID == nil {
		return true, nil
	}
	revision, err := models.FindPullRequestRevision(db, *pullRequest.CurrentRevisionID)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(revision.SHA), strings.TrimSpace(pullRequest.MergeableHeadSHA)), nil
}

func mergeMethodNames(methods []pb.FactoryPullRequestMergeability_MergeMethod) string {
	names := make([]string, 0, len(methods))
	for _, method := range methods {
		switch method {
		case pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH:
			names = append(names, "SQUASH")
		case pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE:
			names = append(names, "MERGE")
		case pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE:
			names = append(names, "REBASE")
		}
	}
	return strings.Join(names, ",")
}

func mergeMethodsFromNames(names []string) []pb.FactoryPullRequestMergeability_MergeMethod {
	methods := make([]pb.FactoryPullRequestMergeability_MergeMethod, 0, len(names))
	for _, name := range names {
		switch strings.ToUpper(strings.TrimSpace(name)) {
		case "SQUASH":
			methods = append(methods, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH)
		case "MERGE":
			methods = append(methods, pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE)
		case "REBASE":
			methods = append(methods, pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE)
		}
	}
	return methods
}

func mergeabilityBlockedReasonName(reason pb.FactoryPullRequestMergeability_BlockedReason) string {
	switch reason {
	case pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN:
		return "ACTIVE_RUN"
	case pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED:
		return "CHECKS_UNFINISHED"
	case pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED:
		return "CHECK_FAILED"
	case pb.FactoryPullRequestMergeability_BLOCKED_REASON_DRAFT:
		return "DRAFT"
	case pb.FactoryPullRequestMergeability_BLOCKED_REASON_CONFLICTING:
		return "CONFLICTING"
	case pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION:
		return "MISSING_INTEGRATION"
	default:
		return ""
	}
}

func mergeabilityBlockedReasonFromName(name string) pb.FactoryPullRequestMergeability_BlockedReason {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "ACTIVE_RUN":
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN
	case "CHECKS_UNFINISHED":
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED
	case "CHECK_FAILED":
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED
	case "DRAFT":
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_DRAFT
	case "CONFLICTING":
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_CONFLICTING
	case "MISSING_INTEGRATION":
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION
	default:
		return pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNSPECIFIED
	}
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
