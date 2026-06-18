package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// maxSnoozePolicies is the number of alert policies a single snooze may target
// (Cloud Monitoring's per-snooze limit).
const maxSnoozePolicies = 16

// snoozeDurationOptions are the supported snooze windows, expressed as Go
// durations so the component can compute the interval end time from "now".
var snoozeDurationOptions = []configuration.FieldOption{
	{Label: "1 hour", Value: "1h"},
	{Label: "6 hours", Value: "6h"},
	{Label: "12 hours", Value: "12h"},
	{Label: "1 day", Value: "24h"},
	{Label: "7 days", Value: "168h"},
	{Label: "30 days", Value: "720h"},
}

func isValidSnoozeDuration(v string) bool {
	return optionHasValue(snoozeDurationOptions, v)
}

// snooze models the subset of the Cloud Monitoring Snooze resource the
// components read back after create/get/update.
type snooze struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Criteria    *struct {
		Policies []string `json:"policies"`
	} `json:"criteria"`
	Interval *struct {
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
	} `json:"interval"`
}

// snoozePayload normalizes a Snooze into the component output payload.
func snoozePayload(s *snooze) map[string]any {
	payload := map[string]any{
		"name":        s.Name,
		"id":          lastSegment(s.Name),
		"displayName": s.DisplayName,
	}
	if s.Criteria != nil && len(s.Criteria.Policies) > 0 {
		payload["policies"] = s.Criteria.Policies
		payload["policiesCount"] = len(s.Criteria.Policies)
	}
	if s.Interval != nil {
		if s.Interval.StartTime != "" {
			payload["startTime"] = s.Interval.StartTime
		}
		if s.Interval.EndTime != "" {
			payload["endTime"] = s.Interval.EndTime
		}
	}
	return payload
}

// parseSnoozeName extracts the relative resource name from a snooze value. The
// value is `projects/<project>/snoozes/<id>`, optionally as a full
// monitoring.googleapis.com URL.
func parseSnoozeName(value string) (string, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", errors.New("snooze is required")
	}
	idx := strings.Index(s, "projects/")
	if idx < 0 {
		return "", fmt.Errorf("snooze %q must be a resource name like projects/<project>/snoozes/<id>", value)
	}
	rel := s[idx:]
	if q := strings.IndexAny(rel, "?#"); q >= 0 {
		rel = rel[:q]
	}
	rel = strings.TrimRight(rel, "/")
	parts := strings.Split(rel, "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "snoozes" || parts[1] == "" || parts[3] == "" {
		return "", fmt.Errorf("snooze %q is not a valid snooze name", value)
	}
	return rel, nil
}

// resolveSnoozeName parses the selected value into a relative snooze name and
// verifies it belongs to the integration's project, mirroring resolvePolicyName
// so cross-project snooze operations are rejected rather than silently hitting
// another project.
func resolveSnoozeName(value, project string) (string, error) {
	name, err := parseSnoozeName(value)
	if err != nil {
		return "", err
	}
	// parseSnoozeName guarantees the shape projects/<project>/snoozes/<id>.
	urlProject := strings.Split(name, "/")[1]
	if project != "" && urlProject != project {
		return "", fmt.Errorf(
			"snooze belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		)
	}
	return name, nil
}

// snoozeSelectorField is the shared "pick a snooze" field used by Get and Expire.
func snoozeSelectorField() configuration.Field {
	return configuration.Field{
		Name:        "snooze",
		Label:       "Snooze",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The snooze to target. Lists the snoozes in your project.",
		Placeholder: "Select snooze",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeSnooze},
		},
	}
}

func validateSnoozeSelection(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("snooze is required")
	}
	// Expressions are resolved at execution time.
	if strings.Contains(value, "{{") {
		return nil
	}
	_, err := parseSnoozeName(value)
	return err
}

// SnoozeNodeMetadata is stored on the node at Setup so the collapsed UI can show
// the snooze's human-readable display name instead of its numeric ID.
type SnoozeNodeMetadata struct {
	SnoozeName  string `json:"snoozeName" mapstructure:"snoozeName"`
	DisplayName string `json:"displayName" mapstructure:"displayName"`
	ID          string `json:"id" mapstructure:"id"`
}

// resolveSnoozeMetadata best-effort resolves the selected snooze's display name
// via the API and stores it on the node. It falls back to the parsed ID when the
// value is an expression or the API is unavailable, so Setup never fails just
// because the display name could not be fetched. Mirrors resolveAlertPolicyMetadata.
func resolveSnoozeMetadata(ctx core.SetupContext, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	// Expressions are resolved at execution time; store the raw value so the UI
	// still shows something meaningful.
	if strings.Contains(value, "{{") {
		return ctx.Metadata.Set(SnoozeNodeMetadata{SnoozeName: value})
	}

	name, err := parseSnoozeName(value)
	if err != nil {
		return err
	}

	fallback := SnoozeNodeMetadata{SnoozeName: name, ID: lastSegment(name)}
	if ctx.Integration == nil {
		return ctx.Metadata.Set(fallback)
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}

	body, err := client.GetURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, name))
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}
	var s snooze
	if err := json.Unmarshal(body, &s); err != nil {
		return ctx.Metadata.Set(fallback)
	}
	display := s.DisplayName
	if display == "" {
		display = lastSegment(name)
	}
	return ctx.Metadata.Set(SnoozeNodeMetadata{
		SnoozeName:  name,
		DisplayName: display,
		ID:          lastSegment(name),
	})
}
