package github

import (
	"crypto/sha256"
	"encoding/binary"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ManualTaskMarkerPrefix identifies a GitHub issue opened for a manual task.
// The issue body carries the marker so a late webhook can find the task even
// when the create response never arrived.
const ManualTaskMarkerPrefix = "superplane-manual-task:"

const IssueEventPayloadType = "github.issue"

// manualTaskActorSeparator keeps the GitHub login that opened the issue next
// to the marker. The login is not taken from the issue body.
const manualTaskActorSeparator = "\x1f"

func NewManualTaskMarker() string {
	return ManualTaskMarkerPrefix + uuid.NewString()
}

// PendingManualTaskLabel stores the marker and the integration's GitHub login.
// A webhook may attach the issue only when that login matches the issue author.
func PendingManualTaskLabel(marker, actor string) string {
	marker = strings.TrimSpace(marker)
	actor = strings.TrimSpace(actor)
	if marker == "" || actor == "" {
		return marker
	}
	return marker + manualTaskActorSeparator + actor
}

// AppBotLogin returns the GitHub login that opens issues for an app installation.
// A manual-task issue is attached only when its author is this login.
func AppBotLogin(integration *models.Integration) string {
	if integration == nil || integrationProperty(integration, common.PropertyAuthMethod) != common.AuthMethodApp {
		return ""
	}
	slug := integrationProperty(integration, common.PropertyAppSlug)
	if slug == "" {
		return ""
	}
	return slug + "[bot]"
}

func integrationProperty(integration *models.Integration, name string) string {
	for _, property := range integration.Properties {
		if property.Name != name {
			continue
		}
		value, _ := property.Value.(string)
		return strings.TrimSpace(value)
	}
	return ""
}

// ManualTaskActorFromLabel returns the GitHub login stored with a pending marker.
func ManualTaskActorFromLabel(label string) string {
	_, actor, ok := strings.Cut(label, manualTaskActorSeparator)
	if !ok {
		return ""
	}
	return strings.TrimSpace(actor)
}

// ManualTaskMarkerFromLabel returns the marker stored on a pending task.
func ManualTaskMarkerFromLabel(label string) (string, bool) {
	label, _, _ = strings.Cut(strings.TrimSpace(label), manualTaskActorSeparator)
	return ManualTaskMarkerFromBody(label)
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

// IssueAuthorFromEventData returns the GitHub login that opened the issue.
// The login comes from the signed webhook payload, not from the issue body.
func IssueAuthorFromEventData(eventData any) (string, bool) {
	issue, ok := issueObjectFromEventData(eventData)
	if !ok {
		return "", false
	}
	user, ok := issue["user"].(map[string]any)
	if !ok {
		return "", false
	}
	login, _ := user["login"].(string)
	login = strings.TrimSpace(login)
	if login == "" {
		return "", false
	}
	return login, true
}

func issueBodyFromEventData(eventData any) (string, bool) {
	issue, ok := issueObjectFromEventData(eventData)
	if !ok {
		return "", false
	}
	body, _ := issue["body"].(string)
	if strings.TrimSpace(body) == "" {
		return "", false
	}
	return body, true
}

func issueObjectFromEventData(eventData any) (map[string]any, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return nil, false
	}
	if typeName, _ := envelope["type"].(string); typeName != IssueEventPayloadType {
		return nil, false
	}

	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return nil, false
	}
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return nil, false
	}
	return issue, true
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
