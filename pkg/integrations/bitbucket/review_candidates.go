package bitbucket

import (
	"fmt"
	"net/http"
	"net/url"
)

type ReviewCandidate struct {
	ReviewerIdentity `mapstructure:",squash"`
	Evidence         []string `json:"evidence"`
}

type ReviewCandidates struct {
	HeadSHA                   string            `json:"headSha"`
	State                     string            `json:"state"`
	Author                    ReviewerIdentity  `json:"author"`
	Candidates                []ReviewCandidate `json:"candidates"`
	ContributorCommits        int               `json:"contributorCommits"`
	ContributorHistoryLimited bool              `json:"contributorHistoryLimited"`
}

// ListPullRequestReviewCandidates treats defaults and recent commit authors as
// relevance evidence only. Effective access and account status establish eligibility.
func (c *Client) ListPullRequestReviewCandidates(repository string, number int64) (ReviewCandidates, error) {
	result := ReviewCandidates{Candidates: []ReviewCandidate{}}
	snapshot, err := c.GetPullRequestReviewSnapshot(repository, number)
	if err != nil {
		return result, err
	}
	result.HeadSHA, result.State = snapshot.Source.Commit.Hash, snapshot.State
	result.Author = ReviewerIdentity{UUID: bitbucketUUID(snapshot.Author.UUID), DisplayName: snapshot.Author.DisplayName}
	eligible, err := c.ListEligibleReviewers(repository)
	if err != nil {
		return result, err
	}
	path, err := repositoryPath(repository)
	if err != nil {
		return result, err
	}
	defaults, err := readReviewerPages[struct {
		User ReviewerAccount `json:"user"`
	}](c, fmt.Sprintf("%s/repositories/%s/effective-default-reviewers?pagelen=100", baseURL, path))
	if err != nil {
		return result, fmt.Errorf("read effective default reviewers: %w", err)
	}
	defaultReviewers := map[string]bool{}
	for _, record := range defaults {
		id := bitbucketUUID(record.User.UUID)
		if id == "" {
			return result, fmt.Errorf("default reviewer UUID is missing or invalid")
		}
		defaultReviewers[id] = true
	}
	contributors, count, limited, err := c.recentContributors(path)
	if err != nil {
		return result, err
	}
	result.ContributorCommits, result.ContributorHistoryLimited = count, limited
	for _, user := range eligible {
		if user.UUID == result.Author.UUID {
			continue
		}
		candidate := ReviewCandidate{ReviewerIdentity: user, Evidence: []string{}}
		if defaultReviewers[user.UUID] {
			candidate.Evidence = append(candidate.Evidence, "default_reviewer")
		}
		if contributors[user.UUID] {
			candidate.Evidence = append(candidate.Evidence, "recent_contributor")
		}
		result.Candidates = append(result.Candidates, candidate)
	}
	current, err := c.GetPullRequestReviewSnapshot(repository, number)
	if err != nil {
		return result, err
	}
	if current.Source.Commit.Hash != result.HeadSHA || current.State != result.State || bitbucketUUID(current.Author.UUID) != result.Author.UUID {
		return result, fmt.Errorf("pull request changed during candidate discovery; retry discovery")
	}
	return result, nil
}

// Recent history is an explicit 100-commit sample. Follow small provider pages
// until the sample is full; never present it as complete repository ownership.
func (c *Client) recentContributors(path string) (map[string]bool, int, bool, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s/commits?pagelen=100", baseURL, path)
	first, _ := url.Parse(endpoint)
	contributors := map[string]bool{}
	count := 0
	seen := map[string]bool{}
	for endpoint != "" {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != first.Scheme || parsed.Host != first.Host || parsed.Path != first.Path || parsed.User != nil || seen[endpoint] || len(seen) >= 100 {
			return nil, 0, false, fmt.Errorf("invalid Bitbucket contributor pagination")
		}
		seen[endpoint] = true
		var page struct {
			Values *[]struct {
				Author struct {
					User *ReviewerAccount `json:"user"`
				} `json:"author"`
			} `json:"values"`
			Next string `json:"next"`
		}
		if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &page); err != nil {
			return nil, 0, false, fmt.Errorf("read contributor evidence: %w", err)
		}
		if page.Values == nil {
			return nil, 0, false, fmt.Errorf("contributor page has no values")
		}
		for i, commit := range *page.Values {
			count++
			if commit.Author.User != nil {
				id := bitbucketUUID(commit.Author.User.UUID)
				if id == "" {
					return nil, 0, false, fmt.Errorf("linked contributor UUID is missing or invalid")
				}
				contributors[id] = true
			}
			if count == 100 {
				return contributors, count, page.Next != "" || i < len(*page.Values)-1, nil
			}
		}
		if page.Next == "" {
			break
		}
		next, err := url.Parse(page.Next)
		if err != nil {
			return nil, 0, false, err
		}
		endpoint = first.ResolveReference(next).String()
	}
	return contributors, count, false, nil
}
