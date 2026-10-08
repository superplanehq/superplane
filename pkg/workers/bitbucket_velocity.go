package workers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, err := b.worker.bitbucketClient(b.organizationID, b.integrationID)
	if err != nil {
		return nil, err
	}
	return listBitbucketMerges(ctx, client, repository, from, to)
}

// listBitbucketMerges returns the pull requests that merged in [from, to).
// The merge instant is the pull request updated_on. The merge commit is read
// only for the agent co-author trailer. A fast-forward merge reuses an older
// commit, so that commit date is not the merge date.
func listBitbucketMerges(
	ctx context.Context,
	client *bitbucketintegration.Client,
	repository string,
	from, to time.Time,
) ([]vcs.MergedPullRequest, error) {
	prs, truncated, err := client.ListMergedPullRequests(repository, from, velocitySyncMaxPages)
	if err != nil {
		return nil, err
	}
	if truncated {
		return nil, fmt.Errorf("bitbucket merge list exceeded %d pages before the window start", velocitySyncMaxPages)
	}

	merged := make([]vcs.MergedPullRequest, 0, len(prs))
	for i := range prs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		merge, ok, err := toBitbucketRepositoryMerge(client, repository, &prs[i], from, to)
		if err != nil {
			return nil, err
		}
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
) (vcs.MergedPullRequest, bool, error) {
	if pr == nil || pr.ID <= 0 {
		return vcs.MergedPullRequest{}, false, nil
	}

	// The commit message classifies agent output. Its date is not the merge
	// instant: a fast-forward merge reuses a commit that can predate the pull
	// request.
	mergedAt := pr.UpdatedOn
	agent := false
	if pr.MergeHash != "" {
		commit, err := client.GetMergeCommit(repository, pr.MergeHash)
		if err != nil {
			if !bitbucketCommitMissing(err) {
				return vcs.MergedPullRequest{}, false, fmt.Errorf("read merge commit %s: %w", pr.MergeHash, err)
			}
		} else if commit != nil {
			agent = hasAgentCoAuthor(commit.Message)
		}
	}
	if mergedAt.IsZero() || mergedAt.Before(from) || !mergedAt.Before(to) {
		return vcs.MergedPullRequest{}, false, nil
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
	return merge, true, nil
}

// bitbucketCommitMissing reports a permanent missing commit. Temporary API
// and transport failures must fail the sync so a later retry keeps the
// stored rows.
func bitbucketCommitMissing(err error) bool {
	var apiErr *bitbucketintegration.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
