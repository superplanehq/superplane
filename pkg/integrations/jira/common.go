package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func unmarshalWebhookPayloads[T any](body []byte) ([]T, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	if body[0] == '[' {
		var payloads []T
		if err := json.Unmarshal(body, &payloads); err != nil {
			return nil, err
		}
		return payloads, nil
	}

	var payload T
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return []T{payload}, nil
}

// NodeMetadata stores metadata on action component nodes.
type NodeMetadata struct {
	Project   *Project `json:"project,omitempty"`
	IssueType string   `json:"issueType,omitempty"`
	Status    string   `json:"status,omitempty"`
}

func siteURLFromIntegration(integration core.IntegrationContext) string {
	if integration == nil {
		return ""
	}
	return SiteURLFromMetadata(integration.GetMetadata())
}

func SiteURLFromMetadata(raw any) string {
	if raw == nil {
		return ""
	}
	metadata := Metadata{}
	if err := mapstructure.Decode(raw, &metadata); err != nil {
		return ""
	}
	return strings.TrimSpace(metadata.SiteURL)
}

func requireProject(httpCtx core.HTTPContext, integration core.IntegrationContext, projectKey string) (*Project, error) {
	if httpCtx != nil {
		client, err := NewClient(httpCtx, integration)
		if err == nil {
			projects, err := client.ListProjects()
			if err == nil {
				return findProject(projects, projectKey)
			}
		}
	}

	return requireProjectFromMetadata(integration, projectKey)
}

func requireProjectFromMetadata(integration core.IntegrationContext, projectKey string) (*Project, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	return findProject(metadata.Projects, projectKey)
}

func findProject(projects []Project, projectKey string) (*Project, error) {
	for _, project := range projects {
		if project.Key == projectKey {
			p := project
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", projectKey)
}

func cloudIDFromIntegration(integration core.IntegrationContext) (string, error) {
	meta := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &meta); err != nil {
		return "", fmt.Errorf("decode integration metadata: %w", err)
	}
	if meta.CloudID == "" {
		return "", fmt.Errorf("integration is missing cloud id; re-sync the Jira integration after upgrading SuperPlane")
	}
	return meta.CloudID, nil
}

// applyStatus moves an issue to the requested status. It looks up available
// transitions from the issue's current state and executes the one whose target
// status name matches. Returns an error if no such transition exists.
func applyStatus(client *Client, issueKey, status string) error {
	return applyStatusWithOptions(client, issueKey, status, DoTransitionOptions{})
}

// applyStatusWithOptions looks up the transitions reachable from the issue's
// current state, picks the best one whose target status matches, and runs it.
//
// When a Resolution is requested, the picker prefers a transition whose
// screen actually exposes the resolution field. Jira returns
//
//	{"errors":{"resolution":"Field 'resolution' cannot be set. It is not on the appropriate screen, or unknown."}}
//
// when you set `fields.resolution` on a transition whose screen has no
// resolution field. Pre-filtering against transition.Fields avoids that 400.
// If no matching transition has resolution on its screen, return a clear
// error so the user can either drop the resolution or configure the
// workflow's transition screen.
func applyStatusWithOptions(client *Client, issueKey, status string, opts DoTransitionOptions) error {
	return applyMatchedStatus(client, issueKey, status, opts, matchTransitionsByName(status), false)
}

// ApplyCompletionStatus moves an issue to the chosen column, or to a reachable
// Done-category status when column is empty. Prefer a status named Done.
func ApplyCompletionStatus(client *Client, issueKey, column string, opts DoTransitionOptions) error {
	column = strings.TrimSpace(column)
	matcher := matchTransitionsByName(column)
	if column == "" {
		matcher = matchDoneCategoryTransitions
		column = "Done"
	}
	return applyMatchedStatus(client, issueKey, column, opts, matcher, true)
}

func applyMatchedStatus(
	client *Client,
	issueKey, status string,
	opts DoTransitionOptions,
	match func([]Transition) []Transition,
	defaultResolution bool,
) error {
	transitions, err := client.GetIssueTransitions(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch transitions: %w", err)
	}

	matches := match(transitions)
	if len(matches) == 0 {
		available := make([]string, 0, len(transitions))
		for _, t := range transitions {
			available = append(available, t.To.Name)
		}
		return fmt.Errorf("no transition available to status %q (available: %v)", status, available)
	}

	if defaultResolution && strings.TrimSpace(opts.Resolution) == "" && anyTransitionHasField(matches, "resolution") {
		opts.Resolution = defaultResolutionName(client)
	}

	// Resolution and comment differ in how strictly Jira gates them:
	//   - Resolution is sent via `fields`, which Jira rejects outright unless
	//     the field is on the transition's screen — so it's a hard requirement.
	//   - A comment is sent via `update.comment`, which Jira generally accepts
	//     even when the screen metadata doesn't list a comment field — so it's
	//     only a soft preference; requiring it would block otherwise-valid moves.
	wantsResolution := strings.TrimSpace(opts.Resolution) != ""
	wantsComment := strings.TrimSpace(opts.Comment) != ""

	// First choice: a transition whose screen exposes everything we want to set,
	// so the comment lands atomically with the transition when possible.
	for _, t := range matches {
		if (!wantsResolution || t.HasField("resolution")) && (!wantsComment || t.HasField("comment")) {
			return client.DoTransitionWithOptions(issueKey, t.ID, opts)
		}
	}

	// Otherwise fall back to any transition that accepts the resolution (the
	// hard requirement); the comment is still attached and Jira usually accepts
	// it even without a dedicated comment field on the screen.
	for _, t := range matches {
		if !wantsResolution || t.HasField("resolution") {
			return client.DoTransitionWithOptions(issueKey, t.ID, opts)
		}
	}

	// No matching transition exposes the resolution field. Surface a clear
	// error instead of letting Jira's confusing "not on the appropriate screen"
	// message bubble up.
	names := make([]string, 0, len(matches))
	for _, t := range matches {
		names = append(names, t.Name)
	}
	return fmt.Errorf(
		"transition to %q does not allow setting a resolution; configure the resolution field on the transition screen for %v in Jira, or leave Resolution empty",
		status, names,
	)
}

func matchTransitionsByName(status string) func([]Transition) []Transition {
	return func(transitions []Transition) []Transition {
		var matches []Transition
		for _, t := range transitions {
			if strings.EqualFold(t.To.Name, status) {
				matches = append(matches, t)
			}
		}
		return matches
	}
}

func matchDoneCategoryTransitions(transitions []Transition) []Transition {
	var namedDone []Transition
	var anyDone []Transition
	for _, t := range transitions {
		if !isDoneCategory(t.To.Category) {
			continue
		}
		anyDone = append(anyDone, t)
		if strings.EqualFold(strings.TrimSpace(t.To.Name), "Done") {
			namedDone = append(namedDone, t)
		}
	}
	if len(namedDone) > 0 {
		return namedDone
	}
	return anyDone
}

func anyTransitionHasField(transitions []Transition, fieldID string) bool {
	for _, t := range transitions {
		if t.HasField(fieldID) {
			return true
		}
	}
	return false
}

func defaultResolutionName(client *Client) string {
	resolutions, err := client.ListResolutions()
	if err != nil || len(resolutions) == 0 {
		return ""
	}
	for _, preferred := range []string{"Done", "Fixed", "Resolved"} {
		for _, resolution := range resolutions {
			if strings.EqualFold(resolution.Name, preferred) {
				return resolution.Name
			}
		}
	}
	return resolutions[0].Name
}

// IssueStatusName reads the current workflow status name of an issue.
func IssueStatusName(issue *Issue) string {
	status := issueStatusMap(issue)
	name, _ := status["name"].(string)
	return strings.TrimSpace(name)
}

// IssueStatusCategory reads the current workflow status category of an issue.
func IssueStatusCategory(issue *Issue) string {
	status := issueStatusMap(issue)
	return statusCategoryFromValue(status["statusCategory"])
}

func issueStatusMap(issue *Issue) map[string]any {
	if issue == nil || issue.Fields == nil {
		return nil
	}
	status, _ := issue.Fields["status"].(map[string]any)
	return status
}

func IssueAlreadyInColumn(issue *Issue, column string) bool {
	column = strings.TrimSpace(column)
	if column != "" {
		return strings.EqualFold(IssueStatusName(issue), column)
	}
	return isDoneCategory(IssueStatusCategory(issue))
}

// ParseJiraDateTime parses timestamps returned by Jira APIs.
func ParseJiraDateTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
