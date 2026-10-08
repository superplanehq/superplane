package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
)

// bitbucketVelocity lists merged pull requests through the factory Git host
// seam. Mergeability, merge, and close use the Bitbucket provider in the
// factories actions package. This type only lists.
type bitbucketVelocity struct {
	vcs.Unsupported
	worker         *FactoryVelocitySyncWorker
	organizationID uuid.UUID
	integrationID  uuid.UUID
}

func (b *bitbucketVelocity) ListMergedPullRequests(
	ctx context.Context,
	repository string,
	from, to time.Time,
) ([]vcs.MergedPullRequest, error) {
	client, err := b.worker.bitbucketClient(b.organizationID, b.integrationID)
	if err != nil {
		return nil, err
	}
	return listBitbucketMerges(client, repository, from, to)
}

// listBitbucketMerges returns the pull requests that merged in [from, to).
// The merge instant comes from the merge commit, which dates the merge
// itself; updated_on is only the fallback. Agent output is recognized by the
// co-author trailer on the merge commit, matching the GitHub classification.
func listBitbucketMerges(
	client *bitbucketintegration.Client,
	repository string,
	from, to time.Time,
) ([]vcs.MergedPullRequest, error) {
	prs, _, err := client.ListMergedPullRequests(repository, from, velocitySyncMaxPages)
	if err != nil {
		return nil, err
	}

	merged := make([]vcs.MergedPullRequest, 0, len(prs))
	for i := range prs {
		merge, ok := toBitbucketRepositoryMerge(client, repository, &prs[i], from, to)
		if ok {
			merged = append(merged, merge)
		}
	}
	return merged, nil
}

func toBitbucketRepositoryMerge(
	client *bitbucketintegration.Client,
	repository string,
	pr *bitbucketintegration.MergedBitbucketPullRequest,
	from, to time.Time,
) (vcs.MergedPullRequest, bool) {
	if pr == nil || pr.ID <= 0 {
		return vcs.MergedPullRequest{}, false
	}

	// One commit read dates the merge and classifies agent output.
	mergedAt := pr.UpdatedOn
	agent := false
	if commit, err := client.GetMergeCommit(repository, pr.MergeHash); err == nil && commit != nil {
		if !commit.Date.IsZero() {
			mergedAt = commit.Date
		}
		agent = hasAgentCoAuthor(commit.Message)
	}
	if mergedAt.IsZero() || mergedAt.Before(from) || !mergedAt.Before(to) {
		return vcs.MergedPullRequest{}, false
	}

	merge := vcs.MergedPullRequest{
		Repository:      repository,
		Number:          pr.ID,
		Source:          models.FactoryVelocityMergeSourcePeople,
		AuthorLogin:     pr.AuthorNick,
		AuthorUUID:      pr.AuthorUUID,
		AuthorName:      pr.AuthorName,
		AuthorAvatarURL: pr.AuthorAvatar,
		MergedAt:        mergedAt,
	}

	if agent {
		merge.Source = models.FactoryVelocityMergeSourceAgent
	}
	return merge, true
}
