package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateAnnotation struct{}

type CreateAnnotationSpec struct {
	Text         string   `json:"text" mapstructure:"text"`
	Tags         []string `json:"tags" mapstructure:"tags"`
	DashboardUID string   `json:"dashboardUID" mapstructure:"dashboardUID"`
	PanelID      *int64   `json:"panelId,omitempty" mapstructure:"panelId"`
	Time         string   `json:"time" mapstructure:"time"`
	TimeEnd      string   `json:"timeEnd" mapstructure:"timeEnd"`
}

type CreateAnnotationOutput struct {
	ID int64 `json:"id"`
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

- **Text**: The annotation message (required)
- **Tags**: Optional list of tags to label the annotation (e.g. deploy, rollback, incident)
- **Dashboard UID**: Optional UID of the dashboard to scope the annotation to
- **Panel ID**: Optional panel ID within the dashboard to attach the annotation to
- **Time**: Optional start time (defaults to now if omitted)
- **Time End**: Optional end time — providing this creates a region annotation spanning the window

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
			Description: "Labels to attach to the annotation",
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
			Name:        "dashboardUID",
			Label:       "Dashboard UID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Scope the annotation to a specific dashboard (optional)",
		},
		{
			Name:        "panelId",
			Label:       "Panel ID",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Scope the annotation to a specific panel within the dashboard (optional)",
		},
		{
			Name:        "time",
			Label:       "Time",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Annotation start time (defaults to now if omitted)",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: grafanaDateTimeFormat,
				},
			},
		},
		{
			Name:        "timeEnd",
			Label:       "Time End",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Annotation end time — providing this creates a region annotation",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: grafanaDateTimeFormat,
				},
			},
		},
	}
}

func (c *CreateAnnotation) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateCreateAnnotationSpec(spec)
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
		spec.PanelID,
		timeMS,
		timeEndMS,
	)
	if err != nil {
		return fmt.Errorf("error creating annotation: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.annotation.created",
		[]any{CreateAnnotationOutput{ID: id}},
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
	return nil
}

func validateAnnotationTimeRangeMS(timeMS, timeEndMS int64) error {
	if timeMS > 0 && timeEndMS > 0 && timeEndMS < timeMS {
		return errors.New("timeEnd must be at or after time")
	}
	return nil
}

// parseAnnotationTime accepts RFC3339, RFC3339Nano, or local wall time "2006-01-02T15:04".
func parseAnnotationTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.ParseInLocation(grafanaDateTimeFormat, s, time.Local)
}
