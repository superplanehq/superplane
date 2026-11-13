package bitbucket

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
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
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"source"`
	ETag string `json:"-"`
}

type ReviewerRequestOutcome struct {
	Status         string             `json:"status"`
	Reason         string             `json:"reason,omitempty"`
	HeadSHA        string             `json:"headSha"`
	Requested      []ReviewerIdentity `json:"requested"`
	AlreadyCovered []ReviewerIdentity `json:"alreadyCovered"`
	// Unverifiable participant approvals are reported, never promoted to evidence.
	UnresolvedApprovals []ReviewerIdentity `json:"unresolvedApprovals"`
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
	endpoint, err := reviewPullRequestURL(repository, number)
	if err != nil {
		return nil, err
	}
	var snapshot PullRequestReviewSnapshot
	headers, err := c.doJSONWithHeaders(http.MethodGet, endpoint, nil, http.StatusOK, &snapshot, nil)
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

// readReviewerPages rejects incomplete or untrusted pagination instead of
// returning a partial candidate set or forwarding credentials to another host.
func readReviewerPages[T any](c *Client, endpoint string) ([]T, error) {
	result := []T{}
	seen := map[string]bool{}
	first, _ := url.Parse(endpoint)
	for endpoint != "" {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != first.Scheme || parsed.Host != first.Host || parsed.Path != first.Path || parsed.User != nil || parsed.Fragment != "" {
			return nil, fmt.Errorf("invalid Bitbucket reviewer pagination URL")
		}
		if seen[endpoint] || len(seen) >= 1000 {
			return nil, fmt.Errorf("Bitbucket reviewer pagination did not complete")
		}
		seen[endpoint] = true
		var page struct {
			Values *[]T   `json:"values"`
			Next   string `json:"next"`
		}
		if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &page); err != nil {
			return nil, err
		}
		if page.Values == nil {
			return nil, fmt.Errorf("Bitbucket reviewer page has no values")
		}
		result = append(result, (*page.Values)...)
		if page.Next == "" {
			break
		}
		next, err := url.Parse(page.Next)
		if err != nil {
			return nil, fmt.Errorf("invalid Bitbucket reviewer pagination: %w", err)
		}
		endpoint = first.ResolveReference(next).String()
	}
	return result, nil
}

// ListEligibleReviewers reads effective access, including inherited group grants,
// then checks authoritative account type and status. Permission failures fail closed.
func (c *Client) ListEligibleReviewers(repository string) ([]ReviewerIdentity, error) {
	return c.listEligibleReviewers(repository, nil)
}

// Only selected profiles need revalidation at the reviewer write boundary.
func (c *Client) listEligibleReviewers(repository string, selected []ReviewerIdentity) ([]ReviewerIdentity, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}
	workspace, slug, _ := strings.Cut(path, "/")
	endpoint := fmt.Sprintf("%s/workspaces/%s/permissions/repositories/%s?pagelen=100", baseURL, workspace, slug)
	records, err := readReviewerPages[struct {
		Permission string          `json:"permission"`
		User       ReviewerAccount `json:"user"`
	}](c, endpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot verify effective reviewer access; use repository admin credentials with repository read scope: %w", err)
	}
	accounts := map[string]bool{}
	for _, record := range records {
		if !slices.Contains([]string{"read", "write", "admin"}, record.Permission) {
			return nil, fmt.Errorf("effective reviewer permission is missing or unsupported")
		}
		if !slices.Contains([]string{"write", "admin"}, record.Permission) {
			continue
		}
		id, err := NormalizeAccountID(record.User.UUID)
		if err != nil {
			return nil, fmt.Errorf("effective reviewer permission has an invalid UUID")
		}
		if selected != nil && !slices.ContainsFunc(selected, func(reviewer ReviewerIdentity) bool { return reviewer.UUID == id }) {
			continue
		}
		accounts[id] = true
	}
	ids := make([]string, 0, len(accounts))
	for id := range accounts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	eligible := []ReviewerIdentity{}
	for _, id := range ids {
		var account ReviewerAccount
		endpoint = fmt.Sprintf("%s/users/%s", baseURL, url.PathEscape("{"+id+"}"))
		if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &account); err != nil {
			return nil, fmt.Errorf("cannot verify reviewer account %s; check user read scope: %w", id, err)
		}
		if bitbucketUUID(account.UUID) != id {
			return nil, fmt.Errorf("reviewer account UUID does not match %s", id)
		}
		if account.Type == "app_user" {
			continue
		}
		if account.Type != "user" {
			return nil, fmt.Errorf("cannot verify reviewer account type for %s", id)
		}
		if account.AccountStatus == "" {
			return nil, fmt.Errorf("cannot verify active status for reviewer %s", id)
		}
		if account.AccountStatus != "active" || strings.HasSuffix(strings.ToLower(account.Nickname), "[bot]") {
			continue
		}
		eligible = append(eligible, ReviewerIdentity{UUID: id, DisplayName: account.DisplayName})
	}
	return eligible, nil
}

func normalizeReviewers(reviewers []ReviewerIdentity) ([]ReviewerIdentity, error) {
	if len(reviewers) == 0 {
		return nil, fmt.Errorf("at least one reviewer UUID is required")
	}
	normalized := make([]ReviewerIdentity, 0, len(reviewers))
	seen := map[string]bool{}
	for _, reviewer := range reviewers {
		id, err := NormalizeAccountID(reviewer.UUID)
		if err != nil || seen[id] {
			return nil, fmt.Errorf("reviewers require distinct valid Bitbucket UUIDs")
		}
		reviewer.UUID = id
		normalized = append(normalized, reviewer)
		seen[id] = true
	}
	return normalized, nil
}

func staleReviewReason(snapshot *PullRequestReviewSnapshot, sha string) string {
	if snapshot.State != PullRequestStateOpen {
		return "closed"
	}
	if snapshot.Source.Commit.Hash != sha {
		return "head_mismatch"
	}
	return ""
}

// ensurePullRequestReviewers requires the caller to hold the repository/PR write
// lock for the entire call. Bitbucket does not provide an atomic additive API.
func (c *Client) ensurePullRequestReviewers(repository string, number int64, sha string, desired []ReviewerIdentity) (ReviewerRequestOutcome, error) {
	outcome := ReviewerRequestOutcome{Requested: []ReviewerIdentity{}, AlreadyCovered: []ReviewerIdentity{}, UnresolvedApprovals: []ReviewerIdentity{}}
	desired, err := normalizeReviewers(desired)
	if err != nil {
		return outcome, err
	}
	if strings.TrimSpace(sha) == "" {
		return outcome, fmt.Errorf("expected head SHA is required")
	}
	snapshot, err := c.GetPullRequestReviewSnapshot(repository, number)
	if err != nil {
		return outcome, err
	}
	outcome.HeadSHA = snapshot.Source.Commit.Hash
	if reason := staleReviewReason(snapshot, sha); reason != "" {
		outcome.Status, outcome.Reason = "stale_pr", reason
		return outcome, nil
	}
	preserved := map[string]bool{}
	approvals := map[string]bool{}
	wrote := false
	for attempt := 0; attempt < 3; attempt++ {
		eligible, err := c.listEligibleReviewers(repository, desired)
		if err != nil {
			return outcome, err
		}
		// Eligibility reads can outlive a push or a user edit. This is the last read before PUT.
		snapshot, err = c.GetPullRequestReviewSnapshot(repository, number)
		if err != nil {
			return outcome, err
		}
		outcome.HeadSHA = snapshot.Source.Commit.Hash
		if reason := staleReviewReason(snapshot, sha); reason != "" {
			if wrote {
				return outcome, fmt.Errorf("pull request changed during reviewer update: %s", reason)
			}
			outcome.Status, outcome.Reason = "stale_pr", reason
			return outcome, nil
		}
		for index, reviewer := range desired {
			eligibleIndex := slices.IndexFunc(eligible, func(user ReviewerIdentity) bool { return user.UUID == reviewer.UUID })
			if reviewer.UUID == bitbucketUUID(snapshot.Author.UUID) || eligibleIndex < 0 {
				return outcome, fmt.Errorf("reviewer %s is not an eligible human with write access", reviewer.UUID)
			}
			desired[index] = eligible[eligibleIndex]
		}
		retainReviewers(snapshot, preserved, approvals)
		current := map[string]bool{}
		for _, reviewer := range snapshot.Reviewers {
			current[bitbucketUUID(reviewer.UUID)] = true
		}
		if attempt == 0 {
			for _, reviewer := range desired {
				if current[reviewer.UUID] {
					outcome.AlreadyCovered = append(outcome.AlreadyCovered, reviewer)
				} else {
					outcome.Requested = append(outcome.Requested, reviewer)
				}
			}
		}
		for _, reviewer := range desired {
			preserved[reviewer.UUID] = true
		}
		if containsReviewers(current, preserved) {
			if wrote {
				if err := verifyRetainedApprovals(snapshot, approvals); err != nil {
					return outcome, err
				}
			}
			outcome.Status = "already_covered"
			if wrote {
				outcome.Status = "requested"
			} else {
				outcome.Requested = []ReviewerIdentity{}
				outcome.AlreadyCovered = desired
			}
			outcome.UnresolvedApprovals = unresolvedReviewApprovals(snapshot)
			return outcome, nil
		}
		err = c.replacePullRequestReviewers(repository, snapshot, preserved)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) && slices.Contains([]int{http.StatusConflict, http.StatusPreconditionFailed}, apiErr.StatusCode) {
				continue
			}
			return outcome, fmt.Errorf("reviewer update failed or is uncertain; read the pull request before retrying: %w", err)
		}
		wrote = true
		verified, err := c.GetPullRequestReviewSnapshot(repository, number)
		if err != nil {
			return outcome, fmt.Errorf("cannot verify reviewer update: %w", err)
		}
		if reason := staleReviewReason(verified, sha); reason != "" {
			return outcome, fmt.Errorf("pull request changed during reviewer update: %s", reason)
		}
		verifiedIDs := map[string]bool{}
		for _, reviewer := range verified.Reviewers {
			verifiedIDs[bitbucketUUID(reviewer.UUID)] = true
		}
		if err := verifyRetainedApprovals(verified, approvals); err != nil {
			return outcome, err
		}

		if containsReviewers(verifiedIDs, preserved) {
			outcome.Status = "requested"
			outcome.UnresolvedApprovals = unresolvedReviewApprovals(verified)
			return outcome, nil
		}
		retainReviewers(verified, preserved, approvals)
	}
	return outcome, fmt.Errorf("reviewer update did not converge after 3 attempts; read the pull request before retrying")
}

// replacePullRequestReviewers sends only the intended mutable field.
func (c *Client) replacePullRequestReviewers(repository string, snapshot *PullRequestReviewSnapshot, preserved map[string]bool) error {
	endpoint, err := reviewPullRequestURL(repository, snapshot.ID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(preserved))
	for id := range preserved {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	reviewers := make([]map[string]string, 0, len(ids))
	for _, id := range ids {
		reviewers = append(reviewers, map[string]string{"uuid": "{" + id + "}"})
	}
	headers := http.Header{}
	// Weak validators cannot satisfy HTTP If-Match preconditions.
	if snapshot.ETag != "" && !strings.HasPrefix(snapshot.ETag, "W/") {
		headers.Set("If-Match", snapshot.ETag)
	}
	var response map[string]any
	_, err = c.doJSONWithHeaders(http.MethodPut, endpoint, map[string]any{"reviewers": reviewers}, http.StatusOK, &response, headers)
	return err
}

func retainReviewers(snapshot *PullRequestReviewSnapshot, reviewers map[string]bool, approvals map[string]bool) {
	for _, reviewer := range snapshot.Reviewers {
		reviewers[reviewer.UUID] = true
	}
	for _, participant := range snapshot.Participants {
		if participant.Approved {
			approvals[participant.User.UUID] = true
		}
	}
}

func verifyRetainedApprovals(snapshot *PullRequestReviewSnapshot, approvals map[string]bool) error {
	for id := range approvals {
		if !slices.ContainsFunc(snapshot.Participants, func(participant ReviewParticipant) bool {
			return bitbucketUUID(participant.User.UUID) == id && participant.Approved
		}) {
			return fmt.Errorf("approval changed during reviewer update for %s; reconcile review evidence", id)
		}
	}
	return nil
}

func containsReviewers(current map[string]bool, expected map[string]bool) bool {
	for id := range expected {
		if !current[id] {
			return false
		}
	}
	return true
}

func unresolvedReviewApprovals(snapshot *PullRequestReviewSnapshot) []ReviewerIdentity {
	approvals := []ReviewerIdentity{}
	for _, participant := range snapshot.Participants {
		if participant.Approved {
			approvals = append(approvals, ReviewerIdentity{UUID: bitbucketUUID(participant.User.UUID), DisplayName: participant.User.DisplayName})
		}
	}
	return approvals
}
