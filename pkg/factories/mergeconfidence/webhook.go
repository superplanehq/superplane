package mergeconfidence

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// CreateWebhookEvent applies producer ordering at the factory PR boundary.
func CreateWebhookEvent(tx *gorm.DB, node *models.CanvasNode, event *models.CanvasEvent, payloadType string, payload any) (handled, created bool, err error) {
	identity := webhookIdentity(node, payloadType, payload)
	if identity.Key == "" {
		return false, false, nil
	}
	created, err = node.CreateRootEventUnlessLatestInScope(tx, event, identity.Scope, identity.Key, identity.isRedundantAfter)
	return true, created, err
}

// mergeConfidenceWebhookIdentity identifies the diff that an automatic
// Bitbucket merge confidence start assesses.
type mergeConfidenceWebhookIdentity struct {
	// Scope selects all earlier starts of one pull request.
	Scope models.RootEventScope
	// Key identifies the pull request, the head commit, and the destination branch.
	Key string
	// Diff identifies the head commit and the destination branch.
	Diff string
	// UpdatedAt is the Bitbucket update time of the pull request in the event.
	UpdatedAt time.Time
}

// isRedundantAfter reports whether the newest start for the pull request makes
// this start unnecessary. A start for the same diff is redundant, so a
// description update cannot start a loop. A delayed webhook for an earlier
// pull request state is also redundant. Without this check, an old head could
// start after the current head and hide the current result.
func (m mergeConfidenceWebhookIdentity) isRedundantAfter(latest models.CanvasEvent) bool {
	if latest.DedupKey != nil && *latest.DedupKey == m.Key {
		return true
	}
	envelope, _ := latest.Data.Data().(map[string]any)
	stored := parseBitbucketMergeConfidenceEvent(envelope["data"])
	// Starts from other paths and legacy starts have no key in this format.
	if stored.diff() != "" && stored.diff() == m.Diff {
		return true
	}
	latestUpdatedAt, ok := stored.updatedAt()
	if !ok || m.UpdatedAt.IsZero() {
		return false
	}
	return m.UpdatedAt.Before(latestUpdatedAt)
}

type bitbucketMergeConfidenceEvent struct {
	PullRequest struct {
		ID string `json:"id"`
	} `json:"pullRequest"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	View struct {
		Number json.Number `json:"number"`
		Head   struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Raw struct {
		UpdatedOn string `json:"updated_on"`
	} `json:"pullrequest"`
}

func parseBitbucketMergeConfidenceEvent(payload any) bitbucketMergeConfidenceEvent {
	var event bitbucketMergeConfidenceEvent
	raw, err := json.Marshal(payload)
	if err != nil {
		return bitbucketMergeConfidenceEvent{}
	}
	if err := json.Unmarshal(raw, &event); err != nil {
		return bitbucketMergeConfidenceEvent{}
	}
	return event
}

// diff joins the head commit and the destination branch. Git ref names cannot
// contain ":", so the parts stay unambiguous.
func (e bitbucketMergeConfidenceEvent) diff() string {
	sha := strings.TrimSpace(e.View.Head.SHA)
	if sha == "" {
		return ""
	}
	return sha + ":" + e.View.Base.Ref
}

func (e bitbucketMergeConfidenceEvent) updatedAt() (time.Time, bool) {
	value := strings.TrimSpace(e.Raw.UpdatedOn)
	if value == "" {
		return time.Time{}, false
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return updatedAt, true
}

// Bitbucket emits PR updates for description edits and can deliver webhooks
// late. Skip an automatic merge confidence start when the newest start for the
// PR has the same head and destination branch, or when the newest start
// reflects a later PR update. A retargeted PR gets a new assessment, also when
// it returns to an earlier destination. Explicit reruns use a different
// payload type and remain available.
//
// The newest start can also come from the PR checks start path, which keys
// events by repository, PR number, and head commit, or from an event stored
// before dedup keys existed. The scope includes both, so the rollout of this
// key format does not start a second assessment for the same diff.
func webhookIdentity(node *models.CanvasNode, payloadType string, payload any) mergeConfidenceWebhookIdentity {
	if node == nil || node.Type != models.NodeTypeTrigger || payloadType != "bitbucket.pullRequest" {
		return mergeConfidenceWebhookIdentity{}
	}
	template, _ := node.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if template["id"] != "risk-score" {
		return mergeConfidenceWebhookIdentity{}
	}
	event := parseBitbucketMergeConfidenceEvent(payload)
	diff := event.diff()
	if event.PullRequest.ID == "" || diff == "" {
		return mergeConfidenceWebhookIdentity{}
	}
	scope := models.RootEventScope{KeyPrefix: "merge-confidence:" + event.PullRequest.ID + ":"}
	repository := strings.ToLower(strings.TrimSpace(event.Repository.FullName))
	number := event.View.Number.String()
	if repository != "" && number != "" {
		scope.OtherKeyPrefixes = []string{repository + "|" + number + "|"}
		scope.UnkeyedPayload = []models.EventPayloadMatch{
			{Path: []string{"type"}, Values: []string{"github.pullRequest", "bitbucket.pullRequest"}},
			{Path: []string{"data", "repository", "full_name"}, Values: []string{repository}, IgnoreCase: true},
			{Path: []string{"data", "pull_request", "number"}, Values: []string{number}},
		}
	}
	updatedAt, _ := event.updatedAt()
	return mergeConfidenceWebhookIdentity{
		Scope:     scope,
		Key:       scope.KeyPrefix + diff,
		Diff:      diff,
		UpdatedAt: updatedAt,
	}
}
