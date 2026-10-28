package taskowner

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// commitsPerPath keeps one GitHub request per path. The newest commits
// carry the ranking, because older commits lose most of their weight.
const commitsPerPath = 50

// GitHubHistory reads path history from the GitHub commits API. GitHub
// resolves each commit email to the author account, so the author ID is
// exact.
type GitHubHistory struct {
	Client     *common.Client
	Repository string
}

func (h GitHubHistory) ListPathCommits(ctx context.Context, repositoryPath string, since time.Time) ([]Commit, error) {
	commits, _, err := h.Client.ListCommits(ctx, h.Repository, &github.CommitsListOptions{
		Path:        repositoryPath,
		Since:       since,
		ListOptions: github.ListOptions{PerPage: commitsPerPath},
	})
	if err != nil {
		return nil, fmt.Errorf("list commits for %s: %w", repositoryPath, err)
	}

	result := make([]Commit, 0, len(commits))
	for _, commit := range commits {
		if commit == nil {
			continue
		}
		author := commit.GetAuthor()
		authorID := ""
		if author.GetID() > 0 {
			authorID = strconv.FormatInt(author.GetID(), 10)
		}
		result = append(result, Commit{
			SHA:         commit.GetSHA(),
			AuthorID:    authorID,
			AuthorLogin: author.GetLogin(),
			AuthorIsBot: strings.EqualFold(author.GetType(), "Bot"),
			Message:     commit.GetCommit().GetMessage(),
			CommittedAt: commit.GetCommit().GetAuthor().GetDate().Time,
		})
	}
	return result, nil
}
