package bitbucket

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

const reviewEvidencePrefix = "bitbucket.review-evidence/"
const forgeReviewerStatusEvent = "pullrequest:reviewer_status_updated"

func reviewCheckEvents() []string {
	return []string{"repo:push", pullRequestEventCreated, pullRequestEventUpdated, pullRequestEventApproved, pullRequestEventUnapproved, pullRequestEventFulfilled, pullRequestEventRejected, "pullrequest:changes_request_created", "pullrequest:changes_request_removed"}
}

// ReadApprovalEvidence is shared by reviewer actions and check reconciliation.
// The integration-bound store supplies provider and integration isolation.
func ReadApprovalEvidence(store core.PublicationStore, repository, id string) (map[string]ApprovalEvidence, error) {
	repository = bitbucketUUID(repository)
	if repository == "" {
		return nil, fmt.Errorf("repository UUID is required")
	}
	number, err := parsePullRequestID(id)
	if err != nil {
		return nil, err
	}
	id = fmt.Sprint(number)
	record, err := store.Get(reviewEvidencePrefix + repository + "/" + id)
	if err != nil {
		return nil, err
	}
	evidence := map[string]ApprovalEvidence{}
	if record != nil {
		err = json.Unmarshal(record.Payload, &evidence)
	}
	return evidence, err
}

// Called only after HMAC verification, including the authenticated Forge bridge.
// Events supply evidence and wake reconciliation; they never set check success.
func receiveReviewEvent(store core.PublicationStore, eventKey string, body []byte) error {
	if !slices.Contains(reviewCheckEvents(), eventKey) && eventKey != forgeReviewerStatusEvent {
		return nil
	}
	var event struct {
		Repository struct {
			UUID string `json:"uuid"`
		} `json:"repository"`
		PullRequest struct {
			ID     int64 `json:"id"`
			Source struct {
				Commit struct {
					Hash string `json:"hash"`
				} `json:"commit"`
			} `json:"source"`
		} `json:"pullrequest"`
		Actor    ReviewerIdentity `json:"actor"`
		Approval struct {
			Date time.Time        `json:"date"`
			User ReviewerIdentity `json:"user"`
		} `json:"approval"`
		Timestamp time.Time `json:"timestamp"`
		Reviewers struct {
			Value []struct {
				User      ReviewerIdentity `json:"user"`
				Status    string           `json:"status"`
				Timestamp time.Time        `json:"timestamp"`
			} `json:"value"`
		} `json:"reviewers"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return err
	}
	if _, err := uuid.Parse(event.Repository.UUID); err != nil {
		return fmt.Errorf("review event repository UUID is required")
	}
	repository := bitbucketUUID(event.Repository.UUID)
	id := fmt.Sprint(event.PullRequest.ID)
	if event.PullRequest.ID > 0 && IsValidFullCommitSHA(event.PullRequest.Source.Commit.Hash) {
		user, date := "", time.Time{}
		if eventKey == pullRequestEventApproved && bitbucketUUID(event.Actor.UUID) == bitbucketUUID(event.Approval.User.UUID) {
			user, date = bitbucketUUID(event.Approval.User.UUID), event.Approval.Date
		}
		if eventKey == forgeReviewerStatusEvent {
			for _, reviewer := range event.Reviewers.Value {
				// Forge carries reviewer snapshots. An unchanged old approval in
				// that snapshot cannot acquire the event's newer source commit.
				if reviewer.Status == "approved" && bitbucketUUID(reviewer.User.UUID) == bitbucketUUID(event.Actor.UUID) && reviewer.Timestamp.Equal(event.Timestamp) {
					user, date = bitbucketUUID(reviewer.User.UUID), reviewer.Timestamp
				}
			}
		}
		if user != "" && !date.IsZero() {
			err := store.Update(reviewEvidencePrefix+repository+"/"+id, func(record *core.OrderedPublication) error {
				evidence := map[string]ApprovalEvidence{}
				if len(record.Payload) > 0 {
					if err := json.Unmarshal(record.Payload, &evidence); err != nil {
						return err
					}
				}
				previous := evidence[user]
				if date.Before(previous.Date) {
					return core.ErrPublicationUnchanged
				}
				if date.Equal(previous.Date) {
					if previous.SHA == event.PullRequest.Source.Commit.Hash || previous.Conflicted {
						return core.ErrPublicationUnchanged
					}
					previous.Conflicted = true
					evidence[user] = previous
				} else {
					evidence[user] = ApprovalEvidence{SHA: event.PullRequest.Source.Commit.Hash, Date: date}
				}
				var err error
				record.Payload, err = json.Marshal(evidence)
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	prefix := reviewCheckPrefix + repository + "/"
	if eventKey != "repo:push" {
		if event.PullRequest.ID <= 0 {
			return nil
		}
		prefix += id
	}
	records, err := store.List(prefix)
	if err != nil {
		return err
	}
	for _, record := range records {
		if eventKey != "repo:push" && record.Key != prefix {
			continue
		}
		if !strings.HasPrefix(record.Key, reviewCheckPrefix+repository+"/") {
			continue
		}
		if err := store.Update(record.Key, func(current *core.OrderedPublication) error {
			if current.Disabled {
				return core.ErrPublicationUnchanged
			}
			current.WakeAt = reviewWake(time.Second)
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
