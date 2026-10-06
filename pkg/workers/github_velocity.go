package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
)

// githubVelocity lists merged pull requests through the factory Git host seam.
// Mergeability, merge, and close use the GitHub provider in the factories
// actions package. This type only lists.
type githubVelocity struct {
	vcs.Unsupported
	worker         *FactoryVelocitySyncWorker
	organizationID uuid.UUID
	integrationID  uuid.UUID
}

func (g *githubVelocity) ListMergedPullRequests(
	ctx context.Context,
	repository string,
	from, to time.Time,
) ([]vcs.MergedPullRequest, error) {
	client, err := g.worker.githubClient(g.organizationID, g.integrationID)
	if err != nil {
		return nil, err
	}
	merged, err := listRepositoryMerges(ctx, client, repository, from, to)
	if err != nil {
		return nil, err
	}

	rows := make([]vcs.MergedPullRequest, len(merged))
	for i, merge := range merged {
		rows[i] = vcs.MergedPullRequest{
			Repository:      merge.repository,
			Number:          merge.number,
			Source:          merge.source,
			AuthorLogin:     merge.authorLogin,
			AuthorName:      merge.authorName,
			AuthorAvatarURL: merge.authorAvatarURL,
			MergedAt:        merge.mergedAt,
		}
	}
	return rows, nil
}
