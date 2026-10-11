package factories

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errFactoryBitbucketNotConnected = errors.New("bitbucket is not connected")

const (
	bitbucketMergeBlockedMissingIntegration = "Bitbucket is not connected."
	bitbucketMergeBlockedPermission         = "The Bitbucket user cannot merge this pull request."
	bitbucketMergeFastForwardOnly           = "Bitbucket only allows fast-forward merges on this branch. Merge this pull request in Bitbucket."
)

// bitbucketProvider runs Bitbucket pull request calls for a factory workspace.
// Velocity listing uses its own Bitbucket provider in the sync worker.
type bitbucketProvider struct {
	db      *gorm.DB
	deps    IntakeDependencies
	factory *models.Factory
	client  *bitbucketintegration.Client
}

// ListMergedPullRequests is unused on this client. The velocity worker
// lists merged pull requests through its own Bitbucket provider.
func (b *bitbucketProvider) ListMergedPullRequests(context.Context, string, time.Time, time.Time) ([]vcs.MergedPullRequest, error) {
	return nil, vcs.ErrNotSupported
}

// ReadMergeability reads draft, head, mergeability checks, and build status
// for one pull request. Bitbucket mergeability checks block conflicts,
// insufficient permissions, and required checks. API failures leave the
// pull request disabled with an unavailable status instead of an error.
func (b *bitbucketProvider) ReadMergeability(ctx context.Context, pullRequest *models.FactoryPullRequest) (vcs.Mergeability, error) {
	if err := b.ensureClient(); err != nil {
		if errors.Is(err, errFactoryBitbucketNotConnected) {
			return vcs.Mergeability{
				CanMerge:      false,
				BlockedReason: mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION),
				Message:       bitbucketMergeBlockedMissingIntegration,
			}, nil
		}
		return vcs.Mergeability{}, err
	}

	current, err := b.client.GetPullRequest(pullRequest.Repository, pullRequest.Number)
	if err != nil {
		return unavailableBitbucketMergeability(""), nil
	}
	if current.Draft {
		return vcs.Mergeability{
			CanMerge:      false,
			BlockedReason: mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_DRAFT),
			Message:       mergeBlockedDraft,
			HeadSHA:       strings.TrimSpace(current.SourceHash),
		}, nil
	}

	active, err := factoryPullRequestHasActiveAutomation(b.db, b.factory, pullRequest)
	if err != nil {
		return vcs.Mergeability{}, err
	}
	if active {
		return vcs.Mergeability{
			CanMerge:      false,
			BlockedReason: mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN),
			Message:       mergeBlockedActiveRun,
			HeadSHA:       strings.TrimSpace(current.SourceHash),
		}, nil
	}

	headSHA := strings.TrimSpace(current.SourceHash)
	allowed := bitbucketAllowedMethods(current.DestMergeStrategies)
	if headSHA == "" {
		return vcs.Mergeability{
			CanMerge:       false,
			BlockedReason:  mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED),
			Message:        mergeBlockedChecksUnfinished,
			HeadSHA:        "",
			AllowedMethods: allowed,
		}, nil
	}
	checks, err := b.client.ListMergeabilityChecks(pullRequest.Repository, pullRequest.Number)
	if err != nil {
		return unavailableBitbucketMergeability(headSHA), nil
	}
	if blocked := bitbucketBlockingCheck(checks); blocked != nil {
		return vcs.Mergeability{
			CanMerge:       false,
			BlockedReason:  mergeabilityBlockedReasonName(blocked.Reason),
			Message:        blocked.Message,
			HeadSHA:        headSHA,
			AllowedMethods: allowed,
		}, nil
	}
	statuses, err := b.client.ListCommitStatuses(pullRequest.Repository, headSHA)
	if err != nil {
		return unavailableBitbucketMergeability(headSHA), nil
	}
	builds := bitbucketintegration.NormalizeBuildStatuses(statuses)
	unfinished, failed := false, false
	for _, build := range builds {
		switch {
		case build.Status != bitbucketintegration.BuildStatusCompleted:
			unfinished = true
		case bitbucketintegration.BuildFailed(build):
			failed = true
		}
	}
	switch {
	case failed:
		return vcs.Mergeability{
			CanMerge:       false,
			BlockedReason:  mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED),
			Message:        mergeBlockedCheckFailed,
			HeadSHA:        headSHA,
			AllowedMethods: allowed,
		}, nil
	case unfinished:
		return vcs.Mergeability{
			CanMerge:       false,
			BlockedReason:  mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED),
			Message:        mergeBlockedChecksUnfinished,
			HeadSHA:        headSHA,
			AllowedMethods: allowed,
		}, nil
	default:
		if len(allowed) == 0 {
			return vcs.Mergeability{
				CanMerge:       false,
				BlockedReason:  mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNSPECIFIED),
				Message:        bitbucketMergeFastForwardOnly,
				HeadSHA:        headSHA,
				AllowedMethods: allowed,
			}, nil
		}
		return vcs.Mergeability{
			CanMerge:       true,
			HeadSHA:        headSHA,
			AllowedMethods: allowed,
		}, nil
	}
}

// unavailableBitbucketMergeability disables merging without caching the
// snapshot, so the next read retries the Bitbucket API.
func unavailableBitbucketMergeability(headSHA string) vcs.Mergeability {
	return vcs.Mergeability{
		CanMerge:      false,
		BlockedReason: mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNAVAILABLE),
		Message:       mergeBlockedUnavailable,
		HeadSHA:       headSHA,
	}
}

type bitbucketBlock struct {
	Reason  pb.FactoryPullRequestMergeability_BlockedReason
	Message string
}

// bitbucketBlockingCheck maps the first blocking Bitbucket mergeability
// check to a merge status. Only blocking checks prevent merging; other
// failures are advisory.
func bitbucketBlockingCheck(checks []bitbucketintegration.MergeabilityCheck) *bitbucketBlock {
	for _, check := range checks {
		if !check.Blocking {
			continue
		}
		switch check.Type {
		case bitbucketintegration.MergeabilityCheckGit:
			return &bitbucketBlock{
				Reason:  pb.FactoryPullRequestMergeability_BLOCKED_REASON_CONFLICTING,
				Message: mergeBlockedConflicting,
			}
		case bitbucketintegration.MergeabilityCheckPermission:
			return &bitbucketBlock{
				Reason:  pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNSPECIFIED,
				Message: bitbucketMergeBlockedPermission,
			}
		default:
			return &bitbucketBlock{
				Reason:  pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED,
				Message: mergeBlockedCheckFailed,
			}
		}
	}
	return nil
}

// bitbucketAllowedMethods offers the squash and merge-commit strategies the
// destination branch supports. Fast-forward is never offered as rebase.
func bitbucketAllowedMethods(strategies []string) []string {
	allowed := []string{}
	for _, strategy := range strategies {
		switch strings.ToLower(strings.TrimSpace(strategy)) {
		case "squash":
			allowed = append(allowed, "SQUASH")
		case "merge_commit":
			allowed = append(allowed, "MERGE")
		}
	}
	if len(strategies) == 0 {
		return []string{"SQUASH", "MERGE"}
	}
	return allowed
}

func (b *bitbucketProvider) MergePullRequest(ctx context.Context, pullRequest *models.FactoryPullRequest, method, expectedSHA string) error {
	if err := b.ensureClient(); err != nil {
		return err
	}
	// Recheck mergeability immediately before merging. Bitbucket has no
	// head precondition on the merge request, so a push between this read
	// and the merge can still change the merged commit.
	fresh, err := b.ReadMergeability(ctx, pullRequest)
	if err != nil {
		return err
	}
	if !fresh.CanMerge {
		message := fresh.Message
		if message == "" {
			message = errFactoryPullRequestNotMergeable.Error()
		}
		return errors.New("bitbucket blocked the merge: " + message)
	}
	expectedSHA = strings.TrimSpace(expectedSHA)
	if expectedSHA == "" || !strings.EqualFold(strings.TrimSpace(fresh.HeadSHA), expectedSHA) {
		return errFactoryPullRequestHeadMoved
	}
	if !slices.Contains(fresh.AllowedMethods, bitbucketMergeMethodName(method)) {
		return errFactoryPullRequestMergeMethodNotAllowed
	}
	merged, err := b.client.MergePullRequest(pullRequest.Repository, pullRequest.Number, bitbucketMergeStrategy(method))
	if err != nil {
		return err
	}
	if merged == nil || !strings.EqualFold(strings.TrimSpace(merged.State), bitbucketintegration.PullRequestStateMerged) {
		return errors.New("bitbucket did not confirm the merge")
	}
	return nil
}

// bitbucketMergeStrategy maps a GitHub merge method name to the Bitbucket
// merge strategy. Only squash and merge commit are offered; anything else
// falls back to a merge commit after the allowed-methods check rejects it.
func bitbucketMergeStrategy(method string) string {
	if strings.EqualFold(strings.TrimSpace(method), "squash") {
		return "squash"
	}
	return "merge_commit"
}

// bitbucketMergeMethodName maps a GitHub merge method name to the stored
// Bitbucket method name. Fast-forward is never reported back as rebase.
func bitbucketMergeMethodName(method string) string {
	if strings.EqualFold(strings.TrimSpace(method), "squash") {
		return "SQUASH"
	}
	return "MERGE"
}

func (b *bitbucketProvider) ClosePullRequest(ctx context.Context, ref vcs.PullRequestRef) error {
	if ref.Provider != "" && ref.Provider != models.FactoryPullRequestProviderBitbucket {
		return errCannotClosePullRequest
	}
	if err := b.ensureClient(); err != nil {
		return err
	}
	_, err := b.client.DeclinePullRequest(ref.Repository, ref.Number)
	return err
}

func (b *bitbucketProvider) ensureClient() error {
	if b.client != nil {
		return nil
	}
	client, err := newFactoryBitbucketAPI(b.db, b.deps, b.factory)
	if err != nil {
		return err
	}
	b.client = client
	return nil
}

var newFactoryBitbucketAPI = buildFactoryBitbucketAPI

func buildFactoryBitbucketAPI(db *gorm.DB, deps IntakeDependencies, factory *models.Factory) (*bitbucketintegration.Client, error) {
	if deps.Registry == nil || deps.Encryptor == nil {
		return nil, errFactoryBitbucketNotConnected
	}

	integrationID := strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID)
	if integrationID == "" {
		return nil, errFactoryBitbucketNotConnected
	}

	integration, err := findReadyOnboardingIntegration(db, factory.OrganizationID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFactoryBitbucketNotConnected, err)
	}
	if integration.AppName != models.ProviderBitbucket {
		return nil, errFactoryBitbucketNotConnected
	}

	metadata := bitbucketintegration.Metadata{}
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return nil, fmt.Errorf("%w: %w", errFactoryBitbucketNotConnected, err)
	}

	client, err := bitbucketintegration.NewClient(
		metadata.AuthType,
		deps.Registry.HTTPContextInTransaction(db),
		contexts.NewIntegrationContext(db, nil, integration, deps.Encryptor, deps.Registry, nil),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFactoryBitbucketNotConnected, err)
	}
	return client, nil
}
