package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ApprovalEvidence is written only by authenticated event delivery. Date is the
// provider's approval date, never a comment, ingestion, or run timestamp.
type ApprovalEvidence struct {
	SHA        string    `json:"sha"`
	Date       time.Time `json:"date"`
	Conflicted bool      `json:"conflicted,omitempty"`
}

type ReviewActivity struct {
	Approval *struct {
		Date time.Time        `json:"date"`
		User ReviewerIdentity `json:"user"`
	} `json:"approval,omitempty"`
}

type ReviewerDecision struct {
	UUID   string `json:"uuid"`
	Reason string `json:"reason"`
}

type PullRequestReviewState struct {
	PullRequest *PullRequestReviewSnapshot
	Reviewers   []ReviewerDecision
}

// ListPullRequestReviewActivity reads the complete log. Approval entries do not
// contain a historical source commit, so this operation cannot create evidence.
func (c *Client) ListPullRequestReviewActivity(ctx context.Context, repository string, id int64) ([]ReviewActivity, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("%s/repositories/%s/pullrequests/%d/activity", baseURL, path, id)
	allowed, err := url.Parse(prefix)
	if err != nil {
		return nil, err
	}
	endpoint := prefix + "?pagelen=100"
	seen := map[string]bool{}
	var activities []ReviewActivity
	for endpoint != "" {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != allowed.Scheme || parsed.Host != allowed.Host || parsed.User != nil || !strings.HasSuffix(strings.TrimSuffix(parsed.Path, "/"), fmt.Sprintf("/pullrequests/%d/activity", id)) {
			return nil, fmt.Errorf("invalid Bitbucket activity pagination URL")
		}
		// Next links can use a canonical slug. Keep the requested repository
		// identity and use only the provider's pagination cursor.
		endpoint = prefix + "?" + parsed.RawQuery
		if seen[endpoint] || len(seen) >= 1000 {
			return nil, fmt.Errorf("repeated Bitbucket activity page")
		}
		seen[endpoint] = true
		var page struct {
			Values *[]ReviewActivity `json:"values"`
			Next   string            `json:"next"`
		}
		if err := c.doJSONContext(ctx, http.MethodGet, endpoint, nil, http.StatusOK, &page); err != nil {
			return nil, err
		}
		if page.Values == nil {
			return nil, fmt.Errorf("Bitbucket activity page has no values")
		}
		activities = append(activities, (*page.Values)...)
		endpoint = page.Next
	}
	return activities, nil
}

// ReviewStateRequest supplies integration-scoped evidence for the assessed commit.
type ReviewStateRequest struct {
	Repository  string
	PullNumber  int64
	AssessedSHA string
	Reviewers   []ReviewerIdentity
	Evidence    map[string]ApprovalEvidence
}

// ReadPullRequestReviewState is shared with reviewer selection. Current provider
// state can revoke an approval; only a matching event receipt can attribute it.
func (c *Client) ReadPullRequestReviewState(ctx context.Context, request ReviewStateRequest) (PullRequestReviewState, error) {
	pr, err := c.getPullRequestReviewSnapshot(ctx, request.Repository, request.PullNumber)
	if err != nil {
		return PullRequestReviewState{}, err
	}
	activity, err := c.ListPullRequestReviewActivity(ctx, request.Repository, request.PullNumber)
	if err != nil {
		return PullRequestReviewState{}, err
	}
	latest := map[string]time.Time{}
	for _, entry := range activity {
		if entry.Approval == nil {
			continue
		}
		user := bitbucketUUID(entry.Approval.User.UUID)
		if entry.Approval.Date.After(latest[user]) {
			latest[user] = entry.Approval.Date
		}
	}
	result := PullRequestReviewState{PullRequest: pr}
	for _, reviewer := range request.Reviewers {
		user := bitbucketUUID(reviewer.UUID)
		reason := "approval_missing"
		for _, participant := range pr.Participants {
			if bitbucketUUID(participant.User.UUID) != user {
				continue
			}
			if participant.State == "changes_requested" {
				reason = "changes_requested"
				break
			}
			if !participant.Approved {
				continue
			}
			reason = "approval_commit_unverified"
			receipt, ok := request.Evidence[user]
			if ok && !receipt.Conflicted && receipt.SHA == request.AssessedSHA && pr.Source.Commit.Hash == request.AssessedSHA && !receipt.Date.IsZero() && receipt.Date.Equal(latest[user]) {
				reason = "approved"
			}
		}
		result.Reviewers = append(result.Reviewers, ReviewerDecision{UUID: user, Reason: reason})
	}
	return result, nil
}

type BuildStatusRequest struct {
	Repository  string
	SHA         string
	Refname     string
	URL         string
	State       string
	Description string
}

// PublishBuildStatus replaces the stable key on the specified commit.
func (c *Client) PublishBuildStatus(ctx context.Context, request BuildStatusRequest) error {
	path, err := repositoryPath(request.Repository)
	if err != nil {
		return err
	}
	if !IsValidFullCommitSHA(request.SHA) || request.Refname == "" || request.URL == "" {
		return fmt.Errorf("commit SHA, source branch, and run URL are required")
	}
	if request.State != "INPROGRESS" && request.State != "SUCCESSFUL" && request.State != "FAILED" {
		return fmt.Errorf("invalid Bitbucket review status")
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/commit/%s/statuses/build", baseURL, path, request.SHA)
	var response CommitStatus
	description := string([]rune(request.Description)[:min(len([]rune(request.Description)), 140)])
	return c.doJSONContext(ctx, http.MethodPost, endpoint, map[string]any{"key": reviewStatusKey, "name": "SuperPlane review request", "refname": request.Refname, "url": request.URL, "state": request.State, "description": description}, http.StatusCreated, &response)
}
