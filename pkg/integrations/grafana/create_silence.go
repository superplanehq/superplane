package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type CreateSilence struct{}

type CreateSilenceSpec struct {
	Matchers []SilenceMatcherInput `json:"matchers" mapstructure:"matchers"`
	StartsAt any                   `json:"startsAt" mapstructure:"startsAt"`
	EndsAt   any                   `json:"endsAt" mapstructure:"endsAt"`
	Comment  string                `json:"comment" mapstructure:"comment"`
}

type SilenceMatcherInput struct {
	Name     string `json:"name" mapstructure:"name"`
	Value    string `json:"value" mapstructure:"value"`
	Operator string `json:"operator" mapstructure:"operator"`
	// IsRegex is accepted for older workflows that only had the "Match as Regex" toggle.
	IsRegex bool `json:"isRegex" mapstructure:"isRegex"`
}

type CreateSilenceOutput struct {
	SilenceID  string `json:"silenceId"`
	SilenceURL string `json:"silenceUrl,omitempty"`
}

func (c *CreateSilence) Name() string {
	return "grafana.createSilence"
}

func (c *CreateSilence) Label() string {
	return "Create Silence"
}

func (c *CreateSilence) Description() string {
	return "Create a new silence in the Grafana Alertmanager to suppress alert notifications"
}

func (c *CreateSilence) Documentation() string {
	return `The Create Silence component creates a new Alertmanager silence in Grafana, suppressing alert notifications that match the configured matchers during the specified time window.

## Use Cases

- **Deploy window**: Suppress noisy alerts during a planned maintenance or deployment window
- **Incident management**: Prevent alert storms from flooding on-call channels while an incident is being worked on
- **Testing**: Silence alerts during load tests or chaos experiments

## Configuration

- **Matchers**: One or more label matchers that identify which alerts to silence (required). Each matcher uses an operator: equal (=), not equal (!=), regex match (=~), or regex does not match (!~), matching Grafana Alertmanager semantics.
- **Starts At**: The start of the silence window (required)
- **Ends At**: The end of the silence window (required)
- **Comment**: A description of why the silence is being created (required)
  - The createdBy field sent to Grafana is set automatically to SuperPlane-<org_name> and is not configurable

## Output

Returns the ID of the newly created silence.
`
}

func (c *CreateSilence) Icon() string {
	return "bell-off"
}

func (c *CreateSilence) Color() string {
	return "blue"
}

func (c *CreateSilence) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSilence) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "matchers",
			Label:       "Matchers",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Alert label matchers",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Matcher",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Label Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label name",
							},
							{
								Name:        "value",
								Label:       "Label Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label value",
							},
							{
								Name:        "operator",
								Label:       "Operator",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Default:     "=",
								Description: "How the value is compared to the label (same operators as Grafana Alertmanager silences)",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Equal", Value: "="},
											{Label: "Not equal", Value: "!="},
											{Label: "Regex matches", Value: "=~"},
											{Label: "Regex does not match", Value: "!~"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "startsAt",
			Label:       "Starts At",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-04-07T15:04",
			Description: "Silence start time.\n\nSupports expressions and expects an ISO 8601 / RFC3339 time.\n\nAlso accepts relative values like now+5h.\n\nExamples:\n2026-04-07T15:04\n2026-04-07T15:04:05Z\nnow+5h\n{{$.maintenance_start}}",
		},
		{
			Name:        "endsAt",
			Label:       "Ends At",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "2026-04-07T16:04",
			Description: "Silence end time.\n\nSupports expressions and expects an ISO 8601 / RFC3339 time.\n\nAlso accepts relative values like now+6h.\n\nExamples:\n2026-04-07T16:04\n2026-04-07T16:04:05Z\nnow+6h\n{{$.maintenance_end}}",
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Reason for the silence",
		},
	}
}

func (c *CreateSilence) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateCreateSilenceSpec(spec)
}

func (c *CreateSilence) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateSilenceSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateCreateSilenceSpec(spec); err != nil {
		return err
	}

	startsAt, err := parseSilenceInstant(spec.StartsAt)
	if err != nil {
		return fmt.Errorf("invalid startsAt %s: %w", formatSilenceInstant(spec.StartsAt), err)
	}

	endsAt, err := parseSilenceInstant(spec.EndsAt)
	if err != nil {
		return fmt.Errorf("invalid endsAt %s: %w", formatSilenceInstant(spec.EndsAt), err)
	}

	if !endsAt.After(startsAt) {
		return fmt.Errorf("endsAt must be after startsAt")
	}

	matchers := make([]SilenceMatcher, 0, len(spec.Matchers))
	for _, m := range spec.Matchers {
		matchers = append(matchers, silenceMatcherFromInput(m))
	}

	createdBy := buildSilenceCreatedBy(ctx.OrganizationID)

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	silenceID, err := client.CreateSilence(
		matchers,
		startsAt.UTC().Format(time.RFC3339),
		endsAt.UTC().Format(time.RFC3339),
		strings.TrimSpace(spec.Comment),
		createdBy,
	)
	if err != nil {
		return fmt.Errorf("error creating silence: %w", err)
	}

	silenceURL, _ := buildSilenceWebURL(ctx.Integration, silenceID)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.silence.created",
		[]any{CreateSilenceOutput{SilenceID: silenceID, SilenceURL: silenceURL}},
	)
}

func (c *CreateSilence) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSilence) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSilence) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateSilence) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateSilence) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateSilence) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeCreateSilenceSpec(config any) (CreateSilenceSpec, error) {
	spec := CreateSilenceSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return CreateSilenceSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return CreateSilenceSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateCreateSilenceSpec(spec CreateSilenceSpec) error {
	if len(spec.Matchers) == 0 {
		return errors.New("at least one matcher is required")
	}
	for i, m := range spec.Matchers {
		if strings.TrimSpace(m.Name) == "" {
			return fmt.Errorf("matcher %d: name is required", i+1)
		}
		if strings.TrimSpace(m.Value) == "" {
			return fmt.Errorf("matcher %d: value is required", i+1)
		}
	}
	if isEmptySilenceInstant(spec.StartsAt) {
		return errors.New("startsAt is required")
	}
	if isEmptySilenceInstant(spec.EndsAt) {
		return errors.New("endsAt is required")
	}
	if strings.TrimSpace(spec.Comment) == "" {
		return errors.New("comment is required")
	}
	return nil
}

func isEmptySilenceInstant(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	if t, ok := v.(time.Time); ok {
		return t.IsZero()
	}
	return false
}

func silenceMatcherFromInput(m SilenceMatcherInput) SilenceMatcher {
	name := strings.TrimSpace(m.Name)
	value := strings.TrimSpace(m.Value)
	switch strings.TrimSpace(m.Operator) {
	case "=":
		return SilenceMatcher{Name: name, Value: value, IsRegex: false, IsEqual: true}
	case "!=":
		return SilenceMatcher{Name: name, Value: value, IsRegex: false, IsEqual: false}
	case "=~":
		return SilenceMatcher{Name: name, Value: value, IsRegex: true, IsEqual: true}
	case "!~":
		return SilenceMatcher{Name: name, Value: value, IsRegex: true, IsEqual: false}
	default:
		// Older configs only stored isRegex; positive match was implied.
		return SilenceMatcher{Name: name, Value: value, IsRegex: m.IsRegex, IsEqual: true}
	}
}

var createdBySafeRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
var collapseCreatedByDashesRe = regexp.MustCompile(`-+`)

func buildSilenceCreatedBy(organizationID string) string {
	orgName := ""
	if strings.TrimSpace(organizationID) != "" {
		org, err := models.FindOrganizationByID(organizationID)
		if err == nil && org != nil {
			orgName = org.Name
		}
	}
	return buildSilenceCreatedByFromOrgName(orgName, organizationID)
}

func buildSilenceCreatedByFromOrgName(orgName string, organizationID string) string {
	safeName := sanitizeOrganizationNameForCreatedBy(orgName)
	if safeName == "" {
		safeName = sanitizeOrganizationNameForCreatedBy(organizationID)
	}
	if safeName == "" {
		safeName = "unknown"
	}
	return "SuperPlane-" + safeName
}

func sanitizeOrganizationNameForCreatedBy(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	// Normalize any whitespace sequences into a single dash.
	normalized := strings.Join(strings.Fields(trimmed), "-")
	normalized = createdBySafeRe.ReplaceAllString(normalized, "-")
	normalized = collapseCreatedByDashesRe.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")

	const maxLen = 64
	if len(normalized) > maxLen {
		normalized = normalized[:maxLen]
		normalized = strings.Trim(normalized, "-")
	}

	return normalized
}

// parseSilenceInstant accepts common ISO 8601 / RFC3339 variants, or local wall time
// like "2006-01-02T15:04" (server local TZ).
func parseSilenceInstant(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return time.Time{}, errors.New("empty time")
		}

		if t, ok, err := parseNowInstant(s); ok || err != nil {
			return t, err
		}

		// Formats with explicit timezone (UTC "Z" or an offset)
		timezoneFormats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04Z07:00",
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05.000Z07:00",
		}
		for _, format := range timezoneFormats {
			if t, err := time.Parse(format, s); err == nil {
				return t, nil
			}
		}

		// Local wall time (server local TZ)
		localFormats := []string{
			"2006-01-02T15:04",
			"2006-01-02 15:04",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, format := range localFormats {
			if t, err := time.ParseInLocation(format, s, time.Local); err == nil {
				return t, nil
			}
		}

		return time.Time{}, fmt.Errorf("unsupported time format %q", s)

	default:
		return time.Time{}, fmt.Errorf("unsupported time type %T", value)
	}
}

func parseNowInstant(s string) (time.Time, bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	if normalized == "now" {
		return time.Now().UTC(), true, nil
	}

	if !strings.HasPrefix(normalized, "now") {
		return time.Time{}, false, nil
	}

	// Accept simple relative syntax like: now+5h, now-30m, now + 2h15m
	rest := strings.TrimSpace(normalized[len("now"):])
	if rest == "" {
		return time.Now().UTC(), true, nil
	}

	sign := rest[0]
	if sign != '+' && sign != '-' {
		return time.Time{}, false, nil
	}

	durStr := strings.TrimSpace(rest[1:])
	if durStr == "" {
		return time.Time{}, true, fmt.Errorf("invalid now offset %q", s)
	}

	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return time.Time{}, true, fmt.Errorf("invalid duration %q in %q: %w", durStr, s, err)
	}

	now := time.Now().UTC()
	if sign == '-' {
		return now.Add(-dur), true, nil
	}
	return now.Add(dur), true, nil
}

func formatSilenceInstant(value any) string {
	if s, ok := value.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", value)
}
