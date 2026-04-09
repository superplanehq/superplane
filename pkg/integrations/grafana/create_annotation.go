package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateAnnotation struct{}

type CreateAnnotationSpec struct {
	DashboardUID string   `json:"dashboardUID" mapstructure:"dashboardUID"`
	Panel        string   `json:"panel" mapstructure:"panel"`
	PanelID      *int64   `json:"panelId,omitempty" mapstructure:"panelId"`
	Text         string   `json:"text" mapstructure:"text"`
	Tags         []string `json:"tags" mapstructure:"tags"`
	Time         string   `json:"time" mapstructure:"time"`
	TimeEnd      string   `json:"timeEnd" mapstructure:"timeEnd"`
}

type CreateAnnotationOutput struct {
	ID  int64  `json:"id"`
	URL string `json:"url,omitempty"`
}

func (c *CreateAnnotation) Name() string {
	return "grafana.createAnnotation"
}

func (c *CreateAnnotation) Label() string {
	return "Create Annotation"
}

func (c *CreateAnnotation) Description() string {
	return "Create an annotation in Grafana to mark deploys, incidents, or other operational events on timelines"
}

func (c *CreateAnnotation) Documentation() string {
	return `The Create Annotation component writes an annotation into Grafana, marking operational events on dashboard timelines.

## Use Cases

- **Deploy tracking**: Annotate graphs at the exact moment a deployment is triggered or completes
- **Incident markers**: Place a marker when an incident is opened or resolved for post-incident correlation
- **Maintenance windows**: Mark the start and end of a maintenance window as a region annotation
- **Change correlation**: Record configuration changes, feature flag toggles, or rollbacks directly on the timeline

## Configuration

	- **Dashboard**: Optional — choose a dashboard from your Grafana instance to scope the annotation
	- **Panel**: Required — choose the panel within the selected dashboard to attach the annotation to
	- **Text**: The annotation message (required)
	- **Tags**: Optional list of tags to label the annotation (e.g. deploy, rollback, incident)
	- **Time**: Optional start time value. Examples: ` + "`{{ now() }}`" + ` or ` + "`{{ now() - duration(\"5m\") }}`" + `
	- **Time End**: Optional end time value for a region annotation. Examples: ` + "`{{ now() }}`" + ` or ` + "`{{ now() + duration(\"24h\") }}`" + `

## Output

Returns the ID of the newly created annotation.
`
}

func (c *CreateAnnotation) Icon() string {
	return "bookmark"
}

func (c *CreateAnnotation) Color() string {
	return "blue"
}

func (c *CreateAnnotation) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAnnotation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dashboardUID",
			Label:       "Dashboard",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Scope the annotation to a specific dashboard (optional)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDashboard,
				},
			},
		},
		{
			Name:        "panel",
			Label:       "Panel",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Panel to attach the annotation to",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypePanel,
					Parameters: []configuration.ParameterRef{
						{
							Name: "dashboardUID",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "dashboardUID",
							},
						},
					},
				},
			},
		},
		{
			Name:        "text",
			Label:       "Text",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Annotation message",
			Placeholder: "Deploy v1.2.3 to production",
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Labels to attach",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "time",
			Label:       "Time",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Start time",
			Default:     `{{ now() }}`,
			Placeholder: `{{ now() }}`,
		},
		{
			Name:        "timeEnd",
			Label:       "Time End",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "End time",
			Placeholder: `{{ now() + duration("24h") }}`,
		},
	}
}

func (c *CreateAnnotation) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateCreateAnnotationSpec(spec); err != nil {
		return err
	}

	return setDashboardNodeMetadata(ctx, spec.DashboardUID)
}

func (c *CreateAnnotation) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateCreateAnnotationSpec(spec); err != nil {
		return err
	}

	var timeMS, timeEndMS int64
	panelID, err := resolveAnnotationPanelID(spec.Panel, spec.PanelID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(spec.Time) != "" {
		t, err := parseAnnotationTime(strings.TrimSpace(spec.Time))
		if err != nil {
			return fmt.Errorf("invalid time %q: %w", spec.Time, err)
		}
		timeMS = t.UTC().UnixMilli()
	}

	if strings.TrimSpace(spec.TimeEnd) != "" {
		t, err := parseAnnotationTime(strings.TrimSpace(spec.TimeEnd))
		if err != nil {
			return fmt.Errorf("invalid timeEnd %q: %w", spec.TimeEnd, err)
		}
		timeEndMS = t.UTC().UnixMilli()
	}

	// Region annotations need a start time; default to now when only the end is set.
	if timeEndMS > 0 && timeMS == 0 {
		timeMS = time.Now().UTC().UnixMilli()
	}
	if err := validateAnnotationTimeRangeMS(timeMS, timeEndMS); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	id, err := client.CreateAnnotation(
		strings.TrimSpace(spec.Text),
		spec.Tags,
		strings.TrimSpace(spec.DashboardUID),
		panelID,
		timeMS,
		timeEndMS,
	)
	if err != nil {
		if panelID != nil {
			return fmt.Errorf("error creating annotation for panel %d: %w", *panelID, err)
		}
		return fmt.Errorf("error creating annotation: %w", err)
	}

	output := CreateAnnotationOutput{
		ID: id,
	}
	if strings.TrimSpace(spec.DashboardUID) != "" {
		output.URL = buildAnnotationURL(
			client.buildURL(fmt.Sprintf("/d/%s", url.PathEscape(strings.TrimSpace(spec.DashboardUID)))),
			panelID,
			timeMS,
			timeEndMS,
		)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.annotation.created",
		[]any{output},
	)
}

func (c *CreateAnnotation) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *CreateAnnotation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAnnotation) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateAnnotation) HandleAction(_ core.ActionContext) error {
	return nil
}

func (c *CreateAnnotation) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAnnotation) Cleanup(_ core.SetupContext) error {
	return nil
}

func decodeCreateAnnotationSpec(config any) (CreateAnnotationSpec, error) {
	spec := CreateAnnotationSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return CreateAnnotationSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return CreateAnnotationSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateCreateAnnotationSpec(spec CreateAnnotationSpec) error {
	if strings.TrimSpace(spec.Text) == "" {
		return errors.New("text is required")
	}
	if _, err := resolveAnnotationPanelID(spec.Panel, spec.PanelID); err != nil {
		return err
	}
	if strings.TrimSpace(spec.Panel) == "" && (spec.PanelID == nil || *spec.PanelID <= 0) {
		return errors.New("panel is required")
	}
	return nil
}

func resolveAnnotationPanelID(panel string, legacyPanelID *int64) (*int64, error) {
	if strings.TrimSpace(panel) != "" {
		parsed, err := parseAnnotationPanelID(panel)
		if err != nil {
			return nil, err
		}
		return &parsed, nil
	}

	if legacyPanelID != nil && *legacyPanelID > 0 {
		return legacyPanelID, nil
	}

	return nil, nil
}

func parseAnnotationPanelID(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("panel must be a valid panel resource ID")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("panel must be greater than zero")
	}
	return parsed, nil
}

func validateAnnotationTimeRangeMS(timeMS, timeEndMS int64) error {
	if timeMS > 0 && timeEndMS > 0 && timeEndMS < timeMS {
		return errors.New("timeEnd must be at or after time")
	}
	return nil
}

// parseAnnotationTime accepts Unix milliseconds, RFC3339, RFC3339Nano,
// or relative Grafana values like "now+2h".
func parseAnnotationTime(s string) (time.Time, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return time.Time{}, errors.New("time value is required")
	}

	if t, ok, err := parseRelativeAnnotationTime(trimmed, time.Now().UTC()); err != nil {
		return time.Time{}, err
	} else if ok {
		return t, nil
	}

	if ms, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}

	if t, ok, err := parseGrafanaQueryTime(trimmed, nil); err != nil {
		return time.Time{}, err
	} else if ok {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unsupported time value %q", trimmed)
}

func parseRelativeAnnotationTime(value string, now time.Time) (time.Time, bool, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "now") {
		return time.Time{}, false, nil
	}

	current := now
	remaining := strings.TrimPrefix(trimmed, "now")

	for len(remaining) > 0 {
		switch remaining[0] {
		case '+', '-':
			sign := int64(1)
			if remaining[0] == '-' {
				sign = -1
			}

			amount, unit, rest, err := consumeRelativeAnnotationOffset(remaining[1:])
			if err != nil {
				return time.Time{}, true, err
			}

			current, err = applyRelativeAnnotationOffset(current, sign*amount, unit)
			if err != nil {
				return time.Time{}, true, err
			}

			remaining = rest
		case '/':
			unit, rest, err := consumeRelativeAnnotationRoundingUnit(remaining[1:])
			if err != nil {
				return time.Time{}, true, err
			}

			current, err = roundRelativeAnnotationTime(current, unit)
			if err != nil {
				return time.Time{}, true, err
			}

			remaining = rest
		default:
			return time.Time{}, true, fmt.Errorf("unsupported relative time syntax %q", value)
		}
	}

	return current, true, nil
}

func consumeRelativeAnnotationOffset(input string) (int64, string, string, error) {
	index := 0
	for index < len(input) && input[index] >= '0' && input[index] <= '9' {
		index++
	}
	if index == 0 {
		return 0, "", "", fmt.Errorf("expected relative time amount in %q", input)
	}

	unitStart := index
	for index < len(input) && ((input[index] >= 'a' && input[index] <= 'z') || (input[index] >= 'A' && input[index] <= 'Z')) {
		index++
	}
	if unitStart == index {
		return 0, "", "", fmt.Errorf("expected relative time unit in %q", input)
	}

	amount, err := strconv.ParseInt(input[:unitStart], 10, 64)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid relative time amount %q", input[:unitStart])
	}

	return amount, input[unitStart:index], input[index:], nil
}

func consumeRelativeAnnotationRoundingUnit(input string) (string, string, error) {
	index := 0
	for index < len(input) && ((input[index] >= 'a' && input[index] <= 'z') || (input[index] >= 'A' && input[index] <= 'Z')) {
		index++
	}
	if index == 0 {
		return "", "", fmt.Errorf("expected rounding unit in %q", input)
	}

	return input[:index], input[index:], nil
}

func applyRelativeAnnotationOffset(base time.Time, amount int64, unit string) (time.Time, error) {
	switch unit {
	case "s":
		return base.Add(time.Duration(amount) * time.Second), nil
	case "m":
		return base.Add(time.Duration(amount) * time.Minute), nil
	case "h":
		return base.Add(time.Duration(amount) * time.Hour), nil
	case "d":
		return base.AddDate(0, 0, int(amount)), nil
	case "w":
		return base.AddDate(0, 0, int(amount*7)), nil
	case "M":
		return base.AddDate(0, int(amount), 0), nil
	case "Q":
		return base.AddDate(0, int(amount*3), 0), nil
	case "y", "Y":
		return base.AddDate(int(amount), 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported relative time unit %q", unit)
	}
}

func roundRelativeAnnotationTime(base time.Time, unit string) (time.Time, error) {
	location := base.Location()
	switch unit {
	case "s":
		return base.Truncate(time.Second), nil
	case "m":
		return base.Truncate(time.Minute), nil
	case "h":
		return base.Truncate(time.Hour), nil
	case "d":
		year, month, day := base.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, location), nil
	case "w":
		year, month, day := base.Date()
		start := time.Date(year, month, day, 0, 0, 0, 0, location)
		weekdayOffset := (int(start.Weekday()) + 6) % 7
		return start.AddDate(0, 0, -weekdayOffset), nil
	case "M":
		year, month, _ := base.Date()
		return time.Date(year, month, 1, 0, 0, 0, 0, location), nil
	case "Q":
		year, month, _ := base.Date()
		quarterStartMonth := time.Month(((int(month)-1)/3)*3 + 1)
		return time.Date(year, quarterStartMonth, 1, 0, 0, 0, 0, location), nil
	case "y", "Y":
		year, _, _ := base.Date()
		return time.Date(year, time.January, 1, 0, 0, 0, 0, location), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported rounding unit %q", unit)
	}
}

func buildAnnotationURL(dashboardURL string, panelID *int64, timeMS, timeEndMS int64) string {
	trimmed := strings.TrimSpace(dashboardURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	query := parsed.Query()
	if panelID != nil && *panelID > 0 {
		query.Set("viewPanel", strconv.FormatInt(*panelID, 10))
	}

	if fromMS, toMS, ok := annotationURLTimeRangeMS(timeMS, timeEndMS); ok {
		query.Set("from", strconv.FormatInt(fromMS, 10))
		query.Set("to", strconv.FormatInt(toMS, 10))
	}

	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func annotationURLTimeRangeMS(timeMS, timeEndMS int64) (int64, int64, bool) {
	if timeMS > 0 && timeEndMS > 0 {
		return timeMS, timeEndMS, true
	}

	if timeMS > 0 {
		padding := int64((5 * time.Minute).Milliseconds())
		fromMS := timeMS - padding
		if fromMS < 0 {
			fromMS = 0
		}
		return fromMS, timeMS + padding, true
	}

	return 0, 0, false
}
