package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ReviewerIdentity uses the provider UUID. Names are presentation only.
type ReviewerIdentity struct {
	UUID        string `json:"uuid" mapstructure:"uuid"`
	DisplayName string `json:"displayName,omitempty" mapstructure:"displayName"`
}

type ReviewParticipant struct {
	User     ReviewerIdentity `json:"user"`
	Approved bool             `json:"approved"`
	State    string           `json:"state"`
}

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
	PullRequest *BitbucketPullRequest
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
		if seen[endpoint] {
			return nil, fmt.Errorf("repeated Bitbucket activity page")
		}
		seen[endpoint] = true
		var page struct {
			Values []ReviewActivity `json:"values"`
			Next   string           `json:"next"`
		}
		if err := c.doJSONContext(ctx, http.MethodGet, endpoint, nil, http.StatusOK, &page); err != nil {
			return nil, err
		}
		activities = append(activities, page.Values...)
		endpoint = page.Next
	}
	return activities, nil
}

// ReadPullRequestReviewState is shared with reviewer selection. Current provider
// state can revoke an approval; only a matching event receipt can attribute it.
func (c *Client) ReadPullRequestReviewState(ctx context.Context, repository string, id int64, sha string, reviewers []ReviewerIdentity, evidence map[string]ApprovalEvidence) (PullRequestReviewState, error) {
	pr, err := c.GetPullRequestContext(ctx, repository, id)
	if err != nil {
		return PullRequestReviewState{}, err
	}
	activity, err := c.ListPullRequestReviewActivity(ctx, repository, id)
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
	for _, reviewer := range reviewers {
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
			receipt, ok := evidence[user]
			if ok && !receipt.Conflicted && receipt.SHA == sha && pr.SourceHash == sha && !receipt.Date.IsZero() && receipt.Date.Equal(latest[user]) {
				reason = "approved"
			}
		}
		result.Reviewers = append(result.Reviewers, ReviewerDecision{UUID: user, Reason: reason})
	}
	return result, nil
}

// PublishBuildStatus replaces the stable key on the specified commit.
func (c *Client) PublishBuildStatus(ctx context.Context, repository, sha, refname, resultURL, state, description string) error {
	path, err := repositoryPath(repository)
	if err != nil {
		return err
	}
	if !IsValidFullCommitSHA(sha) || refname == "" || resultURL == "" {
		return fmt.Errorf("commit SHA, source branch, and run URL are required")
	}
	if state != "INPROGRESS" && state != "SUCCESSFUL" && state != "FAILED" {
		return fmt.Errorf("invalid Bitbucket review status")
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/commit/%s/statuses/build", baseURL, path, sha)
	var response CommitStatus
	description = string([]rune(description)[:min(len([]rune(description)), 140)])
	return c.doJSONContext(ctx, http.MethodPost, endpoint, map[string]any{"key": reviewStatusKey, "name": "SuperPlane review request", "refname": refname, "url": resultURL, "state": state, "description": description}, http.StatusCreated, &response)
}
