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

const IssueEventPayloadType = "github.issue"

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
	normalized, ok := normalizeGitHubIssueURL(issueURL)
	if factory == nil || !ok {
		return false, nil
	}

	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Path == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, parsed.Path)
	if err != nil {
		return false, err
	}

	for _, rawURL := range urls {
		found, foundOK := normalizeGitHubIssueURL(rawURL)
		if foundOK && found == normalized {
			return true, nil
		}
	}

	return false, nil
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

	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" || path == "/" {
		return "", false
	}

	return "https://" + host + path, true
}
