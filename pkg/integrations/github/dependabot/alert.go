package dependabot

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// AlertsUnavailableMessage is what SuperPlane says when GitHub refuses the
// Dependabot alerts API. A private repository with alerts turned off, and a
// GitHub App that cannot read them, both look like this.
const AlertsUnavailableMessage = "SuperPlane could not read Dependabot alerts for this repository. Turn on Dependabot alerts and allow the GitHub App to read them."

// AlertRef identifies one Dependabot alert in a repository.
type AlertRef struct {
	Repository string
	Number     int
}

// AlertPayloadType is the canvas event type emitted by github.onDependabotAlert.
const AlertPayloadType = "github.dependabotAlert"

// TaskCopy is the backlog title and description for one alert.
type TaskCopy struct {
	Title       string
	Description string
}

// TaskCopyFromAlert builds the task a factory creates for an alert.
func TaskCopyFromAlert(alert *github.DependabotAlert) TaskCopy {
	if alert == nil {
		return TaskCopy{Title: "Update a vulnerable dependency"}
	}

	name := alertPackageName(alert)
	manifest := ""
	if alert.Dependency != nil {
		manifest = strings.TrimSpace(alert.Dependency.GetManifestPath())
	}
	title := "Bump " + name
	if manifest != "" {
		title += " in " + manifest
	}

	lines := []string{}
	if summary := alertSummary(alert); summary != "" {
		lines = append(lines, summary, "")
	}
	lines = append(lines,
		"Package: "+name+" ("+alertEcosystem(alert)+")",
		"Manifest: "+manifest,
		"Vulnerable versions: "+alertVulnerableRange(alert),
		"Patched version: "+alertPatchedVersion(alert),
		"Severity: "+alertSeverity(alert),
	)
	if page := strings.TrimSpace(alert.GetHTMLURL()); page != "" {
		lines = append(lines, page)
	}

	return TaskCopy{
		Title:       title,
		Description: strings.Join(lines, "\n"),
	}
}

// AlertEvent shapes an API alert like the dependabot_alert webhook body, so
// a seeded item and a received webhook take the same path through the canvas.
func AlertEvent(alert *github.DependabotAlert) (map[string]any, error) {
	encoded, err := json.Marshal(alert)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if err := json.Unmarshal(encoded, &body); err != nil {
		return nil, err
	}
	return map[string]any{
		"action": "created",
		"alert":  body,
	}, nil
}

// AlertRefFromEventData reads the repository and alert number from a canvas
// root event. The envelope looks like:
//
//	{ "type": "github.dependabotAlert", "data": { "alert": { "html_url": "..." } } }
func AlertRefFromEventData(eventData any) (AlertRef, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return AlertRef{}, false
	}
	if typeName, _ := envelope["type"].(string); typeName != AlertPayloadType {
		return AlertRef{}, false
	}
	webhook, ok := envelope["data"].(map[string]any)
	if !ok {
		return AlertRef{}, false
	}
	alert, ok := webhook["alert"].(map[string]any)
	if !ok {
		return AlertRef{}, false
	}
	page, _ := alert["html_url"].(string)
	return AlertRefFromURL(page)
}

// AlertRefFromURL reads owner/repo and the alert number from a Dependabot
// alert page such as https://github.com/acme/payments/security/dependabot/7.
func AlertRefFromURL(rawURL string) (AlertRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return AlertRef{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != "security" || parts[3] != "dependabot" {
		return AlertRef{}, false
	}
	number, err := strconv.Atoi(parts[4])
	if err != nil || number <= 0 || parts[0] == "" || parts[1] == "" {
		return AlertRef{}, false
	}

	return AlertRef{
		Repository: parts[0] + "/" + parts[1],
		Number:     number,
	}, true
}

// Unavailable reports whether GitHub refused the alerts API because alerts
// are off or the app cannot read them.
func Unavailable(err error) bool {
	return common.IsForbiddenError(err)
}

func UnavailableError(err error) error {
	if err == nil {
		return nil
	}
	if Unavailable(err) {
		return fmt.Errorf("%s", AlertsUnavailableMessage)
	}
	return err
}

// LockAlertWorkOrder serializes work-order creation for one alert in this
// factory. The lock is held until the caller commits tx.
func LockAlertWorkOrder(tx *gorm.DB, factory *models.Factory, ref AlertRef) error {
	if tx == nil || factory == nil || ref.Number <= 0 {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", alertLockKey(factory.ID, ref)).Error
}

// AlertHasWorkOrder reports whether this factory already has a work order for
// the alert. Matching covers every work-order state.
func AlertHasWorkOrder(tx *gorm.DB, factory *models.Factory, ref AlertRef) (bool, error) {
	if factory == nil || ref.Number <= 0 || strings.TrimSpace(ref.Repository) == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, "/security/dependabot/"+strconv.Itoa(ref.Number))
	if err != nil {
		return false, err
	}

	for _, rawURL := range urls {
		found, ok := AlertRefFromURL(rawURL)
		if ok && found.Number == ref.Number && strings.EqualFold(found.Repository, ref.Repository) {
			return true, nil
		}
	}

	return false, nil
}

func alertLockKey(factoryID uuid.UUID, ref AlertRef) int64 {
	sum := sha256.New()
	sum.Write([]byte("dependabot-alert-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(strings.ToLower(ref.Repository)))
	sum.Write([]byte(strconv.Itoa(ref.Number)))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

func alertPackageName(alert *github.DependabotAlert) string {
	if alert.Dependency == nil || alert.Dependency.Package == nil {
		return "dependency"
	}
	name := strings.TrimSpace(alert.Dependency.Package.GetName())
	if name == "" {
		return "dependency"
	}
	return name
}

func alertEcosystem(alert *github.DependabotAlert) string {
	if alert.Dependency == nil || alert.Dependency.Package == nil {
		return ""
	}
	return strings.TrimSpace(alert.Dependency.Package.GetEcosystem())
}

func alertSummary(alert *github.DependabotAlert) string {
	if alert.SecurityAdvisory == nil {
		return ""
	}
	return strings.TrimSpace(alert.SecurityAdvisory.GetSummary())
}

func alertSeverity(alert *github.DependabotAlert) string {
	if alert.SecurityAdvisory == nil {
		return ""
	}
	return strings.TrimSpace(alert.SecurityAdvisory.GetSeverity())
}

func alertVulnerableRange(alert *github.DependabotAlert) string {
	if alert.SecurityVulnerability == nil {
		return ""
	}
	return strings.TrimSpace(alert.SecurityVulnerability.GetVulnerableVersionRange())
}

func alertPatchedVersion(alert *github.DependabotAlert) string {
	if alert.SecurityVulnerability == nil || alert.SecurityVulnerability.FirstPatchedVersion == nil {
		return ""
	}
	return strings.TrimSpace(alert.SecurityVulnerability.FirstPatchedVersion.GetIdentifier())
}
