package dependabot

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ErrAlertsDisabled means GitHub refused the alerts API because Dependabot
// alerts are turned off for the repository.
var ErrAlertsDisabled = errors.New("dependabot alerts are disabled")

// ErrAlertsUnreadable means GitHub refused the alerts API because the app
// cannot read them.
var ErrAlertsUnreadable = errors.New("dependabot alerts are not readable")

// AlertsDisabledMessage tells the user to turn Dependabot alerts on.
const AlertsDisabledMessage = "Dependabot alerts are off for this repository. Turn them on in the repository security settings."

// AlertsUnreadableMessage tells the user to allow the GitHub App to read alerts.
const AlertsUnreadableMessage = "The GitHub App cannot read Dependabot alerts. Allow that permission for the app, then try again."

// AlertPayloadType is the canvas event type emitted by github.onDependabotAlert.
const AlertPayloadType = "github.dependabotAlert"

// alertsHeading opens the list of alerts in a package task description.
const alertsHeading = "## Alerts"

// packageFixGuidance tells the agent how to fix the package. It sits in the
// task description, before the alert list, so a later alert can be appended
// and the guidance stays where it is.
const packageFixGuidance = `First check if the package is a direct dependency in the manifest.
If it is direct, update it to the patched version or later.
If it is transitive, find the direct dependency that requires it.
Update that direct dependency to a release that requires the patched version.
Use a version override or resolution only when no such release exists, and say so in the pull request.
Update the lockfile so every listed manifest is fixed.`

// PackageRef identifies one vulnerable package in a repository. GitHub raises
// one alert per advisory per manifest, and the fix for all of them is one
// dependency update, so every alert for the package maps to one task.
type PackageRef struct {
	Repository string
	Ecosystem  string
	Name       string
}

// TaskCopy is the backlog title and description for one package.
type TaskCopy struct {
	Title       string
	Description string
}

// OriginURL is the repository's Dependabot alerts page filtered to the
// package. It is the task origin and the key that finds the open task again.
func (r PackageRef) OriginURL() string {
	query := url.Values{}
	query.Set("q", strings.TrimSpace("is:open package:"+r.Name+" "+ecosystemQualifier(r.Ecosystem)))
	return "https://github.com/" + r.Repository + "/security/dependabot?" + query.Encode()
}

// OriginLabel names the package in the task origin chip.
func (r PackageRef) OriginLabel() string {
	return "Dependabot: " + r.Name
}

// Origin is the work order origin of the package task.
func (r PackageRef) Origin() models.WorkOrderOrigin {
	return models.WorkOrderOrigin{URL: r.OriginURL(), Label: r.OriginLabel()}
}

// Matches reports whether both refs point at the same package. Repository
// names are case-insensitive on GitHub.
func (r PackageRef) Matches(other PackageRef) bool {
	return strings.EqualFold(r.Repository, other.Repository) &&
		strings.EqualFold(r.Ecosystem, other.Ecosystem) &&
		r.Name == other.Name
}

// PackageRefFromAlert reads the package of one API alert.
func PackageRefFromAlert(repository string, alert *github.DependabotAlert) (PackageRef, bool) {
	if alert == nil || alert.Dependency == nil || alert.Dependency.Package == nil {
		return PackageRef{}, false
	}
	return newPackageRef(repository, alert.Dependency.Package.GetEcosystem(), alert.Dependency.Package.GetName())
}

// PackageRefFromEventData reads the package from a canvas root event. The
// envelope looks like:
//
//	{ "type": "github.dependabotAlert", "data": { "alert": { "html_url": "...", "dependency": {...} } } }
func PackageRefFromEventData(eventData any) (PackageRef, bool) {
	alert, ok := alertFromEventData(eventData)
	if !ok {
		return PackageRef{}, false
	}

	page, _ := alert["html_url"].(string)
	repository, ok := repositoryFromAlertURL(page)
	if !ok {
		return PackageRef{}, false
	}

	return newPackageRef(repository, nestedString(alert, "dependency", "package", "ecosystem"), nestedString(alert, "dependency", "package", "name"))
}

// PackageRefFromURL reads the package back out of an origin URL built by
// PackageRef.OriginURL.
func PackageRefFromURL(rawURL string) (PackageRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return PackageRef{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "security" || parts[3] != "dependabot" {
		return PackageRef{}, false
	}

	name, ecosystem := "", ""
	for _, qualifier := range strings.Fields(parsed.Query().Get("q")) {
		if value, ok := strings.CutPrefix(qualifier, "package:"); ok {
			name = value
		}
		if value, ok := strings.CutPrefix(qualifier, "ecosystem:"); ok {
			ecosystem = value
		}
	}

	return newPackageRef(parts[0]+"/"+parts[1], ecosystem, name)
}

// TaskTitle names the package the task fixes. It does not name a manifest or
// say "bump", because the right fix for a transitive package is often an
// update of the direct dependency that pulls it in.
func TaskTitle(ref PackageRef) string {
	return "Fix Dependabot alerts for " + packageLabel(ref)
}

// TaskCopyFromAlerts builds the task for every open alert of one package.
func TaskCopyFromAlerts(ref PackageRef, alerts []*github.DependabotAlert) TaskCopy {
	sections := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		payload, err := alertPayload(alert)
		if err != nil {
			continue
		}
		sections = append(sections, AlertSection(payload))
	}

	return TaskCopy{
		Title:       TaskTitle(ref),
		Description: taskDescription(ref, sections),
	}
}

// AlertSection renders one alert as a block of the task description. The
// canvas description expression writes the same block for the first alert;
// keep the two in step.
func AlertSection(alert map[string]any) string {
	lines := []string{
		"### #" + alertNumber(alert) + " " + nestedString(alert, "security_advisory", "summary"),
		"Severity: " + nestedString(alert, "security_advisory", "severity"),
		"Manifest: " + nestedString(alert, "dependency", "manifest_path"),
		"Vulnerable versions: " + nestedString(alert, "security_vulnerability", "vulnerable_version_range"),
		"Patched version: " + nestedString(alert, "security_vulnerability", "first_patched_version", "identifier"),
	}
	if relationship := nestedString(alert, "dependency", "relationship"); relationship == "direct" || relationship == "transitive" {
		lines = append(lines, "Relationship: "+relationship)
	}
	lines = append(lines, nestedString(alert, "html_url"))
	return strings.Join(lines, "\n")
}

// AlertSectionFromEventData renders the alert carried by a canvas root event.
func AlertSectionFromEventData(eventData any) (string, bool) {
	alert, ok := alertFromEventData(eventData)
	if !ok {
		return "", false
	}
	return AlertSection(alert), true
}

// MergeAlertSection appends one alert block to an existing task description.
// The fix guidance stays at the top, so a new alert goes at the end. A block
// whose alert URL is already in the description is not added twice.
func MergeAlertSection(description, section string) string {
	section = strings.TrimSpace(section)
	if section == "" {
		return description
	}
	if page := lastLine(section); strings.HasPrefix(page, "http") && strings.Contains(description, page) {
		return description
	}

	return strings.TrimRight(description, "\n") + "\n\n" + section
}

// AlertEvent shapes an API alert like the dependabot_alert webhook body, so
// a seeded item and a received webhook take the same path through the canvas.
func AlertEvent(alert *github.DependabotAlert) (map[string]any, error) {
	body, err := alertPayload(alert)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"action": "created",
		"alert":  body,
	}, nil
}

// Unavailable reports whether GitHub refused the alerts API because alerts
// are off or the app cannot read them.
func Unavailable(err error) bool {
	return common.IsForbiddenError(err)
}

// UnavailableError turns a GitHub refusal into a reason the caller can show.
// GitHub uses one status code for both causes and names the cause in the body.
func UnavailableError(err error) error {
	if err == nil || !Unavailable(err) {
		return err
	}
	if strings.Contains(strings.ToLower(githubErrorMessage(err)), "alerts are disabled") {
		return fmt.Errorf("%w", ErrAlertsDisabled)
	}
	return fmt.Errorf("%w", ErrAlertsUnreadable)
}

func githubErrorMessage(err error) string {
	var response *github.ErrorResponse
	if errors.As(err, &response) {
		return response.Message
	}
	var installation *ghinstallation.HTTPError
	if errors.As(err, &installation) {
		return installation.Message
	}
	return ""
}

// LockPackageWorkOrder serializes work-order creation for one package in this
// factory. The lock is held until the caller commits tx.
func LockPackageWorkOrder(tx *gorm.DB, factory *models.Factory, ref PackageRef) error {
	if tx == nil || factory == nil || ref.Name == "" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", packageLockKey(factory.ID, ref)).Error
}

// FindOpenPackageWorkOrder returns the factory's open task for the package,
// or nil. A closed task does not count: a new advisory after the fix starts a
// new task.
func FindOpenPackageWorkOrder(tx *gorm.DB, factory *models.Factory, ref PackageRef) (*models.FactoryWorkOrder, error) {
	if factory == nil || ref.Name == "" || strings.TrimSpace(ref.Repository) == "" {
		return nil, nil
	}

	orders, err := factory.ListWorkOrdersByOriginURLFragment(tx, "/security/dependabot?q=")
	if err != nil {
		return nil, err
	}

	openStates := []string{models.FactoryWorkOrderStateDraft, models.FactoryWorkOrderStateOpen}
	for i := range orders {
		order := &orders[i]
		if order.OriginURL == nil || !slices.Contains(openStates, order.State) {
			continue
		}
		found, ok := PackageRefFromURL(*order.OriginURL)
		if ok && found.Matches(ref) {
			return order, nil
		}
	}

	return nil, nil
}

func newPackageRef(repository, ecosystem, name string) (PackageRef, bool) {
	ref := PackageRef{
		Repository: strings.TrimSpace(repository),
		Ecosystem:  strings.ToLower(strings.TrimSpace(ecosystem)),
		Name:       strings.TrimSpace(name),
	}
	if ref.Repository == "" || ref.Name == "" {
		return PackageRef{}, false
	}
	return ref, true
}

func packageLabel(ref PackageRef) string {
	if ref.Ecosystem == "" {
		return ref.Name
	}
	return ref.Name + " (" + ref.Ecosystem + ")"
}

func taskIntro(ref PackageRef) string {
	return "Fix every open Dependabot alert for " + packageLabel(ref) + "."
}

func taskDescription(ref PackageRef, sections []string) string {
	return taskIntro(ref) + "\n" + packageFixGuidance + "\n\n" + alertsHeading + "\n\n" + strings.Join(sections, "\n\n")
}

func ecosystemQualifier(ecosystem string) string {
	if ecosystem == "" {
		return ""
	}
	return "ecosystem:" + ecosystem
}

func alertPayload(alert *github.DependabotAlert) (map[string]any, error) {
	encoded, err := json.Marshal(alert)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if err := json.Unmarshal(encoded, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func alertFromEventData(eventData any) (map[string]any, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return nil, false
	}
	if typeName, _ := envelope["type"].(string); typeName != AlertPayloadType {
		return nil, false
	}
	webhook, ok := envelope["data"].(map[string]any)
	if !ok {
		return nil, false
	}
	alert, ok := webhook["alert"].(map[string]any)
	return alert, ok
}

// repositoryFromAlertURL reads owner/repo from an alert page such as
// https://github.com/acme/payments/security/dependabot/7.
func repositoryFromAlertURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" || parts[2] != "security" || parts[3] != "dependabot" {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

func alertNumber(alert map[string]any) string {
	switch number := alert["number"].(type) {
	case float64:
		return strconv.Itoa(int(number))
	case int:
		return strconv.Itoa(number)
	case int64:
		return strconv.FormatInt(number, 10)
	case json.Number:
		return number.String()
	default:
		return "0"
	}
}

func nestedString(value map[string]any, path ...string) string {
	current := any(value)
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = next[key]
	}
	text, _ := current.(string)
	return strings.TrimSpace(text)
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func packageLockKey(factoryID uuid.UUID, ref PackageRef) int64 {
	sum := sha256.New()
	sum.Write([]byte("dependabot-package-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(strings.ToLower(ref.Repository)))
	sum.Write([]byte(ref.Ecosystem))
	sum.Write([]byte(ref.Name))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}
