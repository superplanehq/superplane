package bitbucket

import (
	"context"
	"fmt"
	"net/http"
)

// ReviewerIdentity uses canonical, unbraced UUIDs. DisplayName is never identity.
type ReviewerIdentity struct {
	UUID        string `json:"uuid" mapstructure:"uuid"`
	DisplayName string `json:"displayName" mapstructure:"displayName"`
}

type ReviewerAccount struct {
	UUID          string `json:"uuid"`
	DisplayName   string `json:"display_name"`
	Type          string `json:"type"`
	AccountStatus string `json:"account_status"`
	Nickname      string `json:"nickname"`
}

type ReviewParticipant struct {
	User     ReviewerAccount `json:"user"`
	Approved bool            `json:"approved"`
	State    string          `json:"state"`
}

// PullRequestReviewSnapshot is shared with review-state consumers. Approved alone
// has no commit attribution and must never count as current-SHA approval evidence.
// Account UUIDs in snapshots are canonical and unbraced.
type PullRequestReviewSnapshot struct {
	ID           int64               `json:"id"`
	State        string              `json:"state"`
	Author       ReviewerAccount     `json:"author"`
	Reviewers    []ReviewerAccount   `json:"reviewers"`
	Participants []ReviewParticipant `json:"participants"`
	Source       struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"source"`
	ETag string `json:"-"`
}

func reviewPullRequestURL(repository string, number int64) (string, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return "", err
	}
	if number <= 0 {
		return "", fmt.Errorf("pull request ID must be positive")
	}
	return fmt.Sprintf("%s/repositories/%s/pullrequests/%d", baseURL, path, number), nil
}

func (c *Client) GetPullRequestReviewSnapshot(repository string, number int64) (*PullRequestReviewSnapshot, error) {
	return c.getPullRequestReviewSnapshot(context.Background(), repository, number)
}

func (c *Client) getPullRequestReviewSnapshot(ctx context.Context, repository string, number int64) (*PullRequestReviewSnapshot, error) {
	endpoint, err := reviewPullRequestURL(repository, number)
	if err != nil {
		return nil, err
	}
	var snapshot PullRequestReviewSnapshot
	headers, err := c.doJSONWithHeadersContext(ctx, http.MethodGet, endpoint, nil, http.StatusOK, &snapshot, nil)
	if err != nil {
		return nil, err
	}
	if snapshot.ID != number || snapshot.Source.Commit.Hash == "" || snapshot.State == "" || snapshot.Reviewers == nil || snapshot.Participants == nil || bitbucketUUID(snapshot.Author.UUID) == "" {
		return nil, fmt.Errorf("pull request review snapshot is incomplete")
	}
	snapshot.Author.UUID = bitbucketUUID(snapshot.Author.UUID)
	for index := range snapshot.Reviewers {
		id := bitbucketUUID(snapshot.Reviewers[index].UUID)
		if id == "" {
			return nil, fmt.Errorf("existing reviewer UUID is missing or invalid")
		}
		snapshot.Reviewers[index].UUID = id
	}
	for index := range snapshot.Participants {
		id := bitbucketUUID(snapshot.Participants[index].User.UUID)
		if id == "" {
			return nil, fmt.Errorf("review participant UUID is missing or invalid")
		}
		snapshot.Participants[index].User.UUID = id
	}
	snapshot.ETag = headers.Get("ETag")
	return &snapshot, nil
}
