package grafana

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListAnnotations struct{}

type ListAnnotationsSpec struct {
	Tags         []string `json:"tags" mapstructure:"tags"`
	DashboardUID string   `json:"dashboardUID" mapstructure:"dashboardUID"`
	From         string   `json:"from" mapstructure:"from"`
	To           string   `json:"to" mapstructure:"to"`
	Limit        int64    `json:"limit" mapstructure:"limit"`
}

type ListAnnotationsOutput struct {
	Annotations []Annotation `json:"annotations"`
}

func (l *ListAnnotations) Name() string {
	return "grafana.listAnnotations"
}

func (l *ListAnnotations) Label() string {
	return "List Annotations"
}

func (l *ListAnnotations) Description() string {
	return "List Grafana annotations filtered by tag, dashboard, or time range"
}

func (l *ListAnnotations) Documentation() string {
	return `The List Annotations component retrieves annotations from Grafana, optionally filtered by tag, dashboard, or time range.

## Use Cases

- **Audit operational events**: Review recent deploy, incident, or change markers on a timeline
- **Correlate incidents**: Retrieve annotations from around an incident time window for post-incident analysis
- **Workflow branching**: Check for existing markers before creating duplicate annotations

## Configuration

- **Tags**: Filter to annotations matching all of the specified tags (optional)
- **Dashboard**: Optional — filter to annotations on a specific dashboard from your Grafana instance
	- **From / To**: Time range filter as relative Grafana values like ` + "`now-1h`" + ` or absolute values with an explicit timezone like ` + "`2026-04-08T15:30Z`" + ` (optional)
- **Limit**: Maximum number of annotations to return (optional)

## Output

Returns a list of annotation objects including ID, text, tags, time, and dashboard/panel references.
`
}

func (l *ListAnnotations) Icon() string {
	return "bookmark"
}

func (l *ListAnnotations) Color() string {
	return "blue"
}

func (l *ListAnnotations) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListAnnotations) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Filter annotations that have all of these tags",
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
			Label:       "Dashboard",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Filter annotations to a specific dashboard",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeDashboard,
				},
			},
		},
		{
			Name:        "from",
			Label:       "From",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Return annotations at or after this time",
			Placeholder: "now-1h or 2026-04-08T15:30Z",
		},
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Return annotations at or before this time",
			Placeholder: "now or 2026-04-08T17:30Z",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Maximum number of annotations to return",
			Placeholder: "100",
		},
	}
}

func (l *ListAnnotations) Setup(ctx core.SetupContext) error {
	spec, err := decodeListAnnotationsSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return setDashboardNodeMetadata(ctx, spec.DashboardUID)
}

func (l *ListAnnotations) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeListAnnotationsSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	var fromMS, toMS int64

	if strings.TrimSpace(spec.From) != "" {
		t, err := parseAnnotationTime(strings.TrimSpace(spec.From))
		if err != nil {
			return fmt.Errorf("invalid from %q: %w", spec.From, err)
		}
		fromMS = t.UTC().UnixMilli()
	}

	if strings.TrimSpace(spec.To) != "" {
		t, err := parseAnnotationTime(strings.TrimSpace(spec.To))
		if err != nil {
			return fmt.Errorf("invalid to %q: %w", spec.To, err)
		}
		toMS = t.UTC().UnixMilli()
	}

	if err := validateListAnnotationTimeRangeMS(fromMS, toMS); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	annotations, err := client.ListAnnotations(
		spec.Tags,
		strings.TrimSpace(spec.DashboardUID),
		fromMS,
		toMS,
		spec.Limit,
	)
	if err != nil {
		return fmt.Errorf("error listing annotations: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.annotations",
		[]any{ListAnnotationsOutput{Annotations: annotations}},
	)
}

func (l *ListAnnotations) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (l *ListAnnotations) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListAnnotations) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListAnnotations) HandleAction(_ core.ActionContext) error {
	return nil
}

func (l *ListAnnotations) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (l *ListAnnotations) Cleanup(_ core.SetupContext) error {
	return nil
}

func validateListAnnotationTimeRangeMS(fromMS, toMS int64) error {
	if fromMS > 0 && toMS > 0 && toMS < fromMS {
		return errors.New("to must be at or after from")
	}
	return nil
}

func decodeListAnnotationsSpec(config any) (ListAnnotationsSpec, error) {
	spec := ListAnnotationsSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return ListAnnotationsSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return ListAnnotationsSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}
