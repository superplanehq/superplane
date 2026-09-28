package github

import (
	"crypto/sha256"
	"encoding/binary"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ManualTaskMarkerPrefix identifies a GitHub issue opened for a manual task.
// The issue body carries the marker so a late webhook can find the task even
// when the create response never arrived.
const ManualTaskMarkerPrefix = "superplane-manual-task:"

const IssueEventPayloadType = "github.issue"

func NewManualTaskMarker() string {
	return ManualTaskMarkerPrefix + uuid.NewString()
}

func ManualTaskMarkerComment(marker string) string {
	return "<!-- " + marker + " -->"
}

func AppendManualTaskMarker(body, marker string) string {
	comment := ManualTaskMarkerComment(marker)
	body = strings.TrimSpace(body)
	if body == "" {
		return comment
	}
	return body + "\n\n" + comment
}

func ManualTaskMarkerFromBody(body string) (string, bool) {
	index := strings.Index(body, ManualTaskMarkerPrefix)
	if index < 0 {
		return "", false
	}

	rest := body[index:]
	end := strings.IndexAny(rest, " \t\r\n>")
	if end < 0 {
		end = len(rest)
	}
	marker := rest[:end]
	if _, err := uuid.Parse(strings.TrimPrefix(marker, ManualTaskMarkerPrefix)); err != nil {
		return "", false
	}
	return marker, true
}

func ManualTaskMarkerFromEventData(eventData any) (string, bool) {
	body, ok := issueBodyFromEventData(eventData)
	if !ok {
		return "", false
	}
	return ManualTaskMarkerFromBody(body)
}

func issueBodyFromEventData(eventData any) (string, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return "", false
	}
	if typeName, _ := envelope["type"].(string); typeName != IssueEventPayloadType {
		return "", false
	}

	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return "", false
	}
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return "", false
	}
	body, _ := issue["body"].(string)
	if strings.TrimSpace(body) == "" {
		return "", false
	}
	return body, true
}

func IssueURLFromEventData(eventData any) (string, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return "", false
	}
	if typeName, _ := envelope["type"].(string); typeName != IssueEventPayloadType {
		return "", false
	}

	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return "", false
	}
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return "", false
	}
	rawURL, _ := issue["html_url"].(string)
	return normalizeGitHubIssueURL(rawURL)
}

// LockIssueWorkOrder serializes work-order creation for one GitHub issue in
// this factory. The lock is held until the caller commits tx.
func LockIssueWorkOrder(tx *gorm.DB, factory *models.Factory, issueURL string) error {
	if tx == nil || factory == nil {
		return nil
	}
	normalized, ok := normalizeGitHubIssueURL(issueURL)
	if !ok {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", githubIssueLockKey(factory.ID, normalized)).Error
}

func githubIssueLockKey(factoryID uuid.UUID, issueURL string) int64 {
	sum := sha256.New()
	sum.Write([]byte("github-issue-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(issueURL))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// IssueHasWorkOrder reports whether this factory already has a work order
// for the GitHub issue. Matching covers every work-order state.
func IssueHasWorkOrder(tx *gorm.DB, factory *models.Factory, issueURL string) (bool, error) {
	order, err := FindIssueWorkOrder(tx, factory, issueURL)
	if err != nil {
		return false, err
	}
	return order != nil, nil
}

// FindIssueWorkOrder returns the factory work order for this GitHub issue, if
// one exists. Matching covers every work-order state.
func FindIssueWorkOrder(tx *gorm.DB, factory *models.Factory, issueURL string) (*models.FactoryWorkOrder, error) {
	normalized, ok := normalizeGitHubIssueURL(issueURL)
	if tx == nil || factory == nil || !ok {
		return nil, nil
	}

	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Path == "" {
		return nil, nil
	}

	orders, err := factory.ListWorkOrdersByOriginURLFragment(tx, parsed.Path)
	if err != nil {
		return nil, err
	}

	for i := range orders {
		origin := orders[i].Origin()
		if origin == nil {
			continue
		}
		found, foundOK := normalizeGitHubIssueURL(origin.URL)
		if foundOK && found == normalized {
			match := orders[i]
			return &match, nil
		}
	}

	return nil, nil
}

func normalizeGitHubIssueURL(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}

	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	if host != "github.com" {
		return "", false
	}

	path := strings.ToLower(strings.TrimRight(parsed.EscapedPath(), "/"))
	if path == "" || path == "/" {
		return "", false
	}

	return "https://" + host + path, true
}
