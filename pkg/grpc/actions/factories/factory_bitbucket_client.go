package factories

import (
	"context"
	"errors"
	"net/http"
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

// ReadMergeability reads draft, head, and build status for one pull request.
// Bitbucket exposes no conflict flag, so conflicts surface at merge time;
// an empty build list does not block, matching the GitHub contract.
func (b *bitbucketProvider) ReadMergeability(ctx context.Context, pullRequest *models.FactoryPullRequest) (vcs.Mergeability, error) {
	if err := b.ensureClient(); err != nil {
		if errors.Is(err, errFactoryBitbucketNotConnected) {
			return vcs.Mergeability{
				CanMerge:      false,
				BlockedReason: mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION),
				Message:       mergeBlockedMissingIntegration,
			}, nil
		}
		return vcs.Mergeability{}, err
	}

	current, err := b.client.GetPullRequest(pullRequest.Repository, pullRequest.Number)
	if err != nil {
		return vcs.Mergeability{}, err
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

	statuses, err := b.client.ListCommitStatuses(pullRequest.Repository, strings.TrimSpace(current.SourceHash))
	if err != nil {
		return vcs.Mergeability{}, err
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
	headSHA := strings.TrimSpace(current.SourceHash)
	// ponytail: strategies default open client-side; live verification can restrict
	allowed := []string{"SQUASH", "MERGE", "REBASE"}
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
		return vcs.Mergeability{
			CanMerge:       true,
			HeadSHA:        headSHA,
			AllowedMethods: allowed,
		}, nil
	}
}

func (b *bitbucketProvider) MergePullRequest(ctx context.Context, repository string, number int, method, expectedSHA string) error {
	if err := b.ensureClient(); err != nil {
		return err
	}
	current, err := b.client.GetPullRequest(repository, int64(number))
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(current.State), bitbucketintegration.PullRequestStateOpen) {
		return errors.New("the pull request is no longer open")
	}
	// ponytail: read-then-merge head check; Bitbucket has no merge precondition
	if expectedSHA = strings.TrimSpace(expectedSHA); expectedSHA != "" &&
		strings.TrimSpace(current.SourceHash) != "" &&
		!strings.EqualFold(strings.TrimSpace(current.SourceHash), expectedSHA) {
		return errFactoryPullRequestHeadMoved
	}
	_, err = b.client.MergePullRequest(repository, int64(number), method)
	if err != nil {
		var apiErr *bitbucketintegration.APIError
		// ponytail: Bitbucket reports unmergeable heads as 409; the board
		// re-syncs and asks for review. Live verification refines this.
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			return errFactoryPullRequestHeadMoved
		}
		return err
	}
	return nil
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
		return nil, errFactoryBitbucketNotConnected
	}
	if integration.AppName != models.ProviderBitbucket {
		return nil, errFactoryBitbucketNotConnected
	}

	metadata := bitbucketintegration.Metadata{}
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return nil, errFactoryBitbucketNotConnected
	}

	client, err := bitbucketintegration.NewClient(
		metadata.AuthType,
		deps.Registry.HTTPContextInTransaction(db),
		contexts.NewIntegrationContext(db, nil, integration, deps.Encryptor, deps.Registry, nil),
	)
	if err != nil {
		return nil, errFactoryBitbucketNotConnected
	}
	return client, nil
}
