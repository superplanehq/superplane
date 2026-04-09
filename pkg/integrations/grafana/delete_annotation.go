package grafana

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteAnnotation struct{}

type DeleteAnnotationSpec struct {
	AnnotationID string `json:"annotationId" mapstructure:"annotationId"`
}

type DeleteAnnotationOutput struct {
	ID      int64 `json:"id"`
	Deleted bool  `json:"deleted"`
}

func (d *DeleteAnnotation) Name() string {
	return "grafana.deleteAnnotation"
}

func (d *DeleteAnnotation) Label() string {
	return "Delete Annotation"
}

func (d *DeleteAnnotation) Description() string {
	return "Delete an existing Grafana annotation by ID"
}

func (d *DeleteAnnotation) Documentation() string {
	return `The Delete Annotation component removes an annotation from Grafana by ID.

## Use Cases

- **Cleanup incorrect markers**: Remove an annotation that was created with wrong text or tags
- **Automated lifecycle**: Delete temporary markers (e.g. maintenance window start) once the event is complete
- **Idempotent workflows**: Allow re-runs to clean up previously created annotations before re-creating them

## Configuration

- **Annotation**: The annotation to delete, chosen from your Grafana instance (required)

## Output

Returns the annotation ID and a confirmation that the annotation was deleted.
`
}

func (d *DeleteAnnotation) Icon() string {
	return "bookmark"
}

func (d *DeleteAnnotation) Color() string {
	return "blue"
}

func (d *DeleteAnnotation) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteAnnotation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "annotationId",
			Label:       "Annotation",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The annotation to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeAnnotation,
				},
			},
		},
	}
}

func (d *DeleteAnnotation) Setup(ctx core.SetupContext) error {
	spec, err := decodeDeleteAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeleteAnnotationSpec(spec); err != nil {
		return err
	}

	return setAnnotationNodeMetadata(ctx, spec.AnnotationID)
}

func (d *DeleteAnnotation) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeleteAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeleteAnnotationSpec(spec); err != nil {
		return err
	}

	id, err := parseAnnotationIDForExecute(spec.AnnotationID)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteAnnotation(id); err != nil {
		return fmt.Errorf("error deleting annotation: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.annotation.deleted",
		[]any{DeleteAnnotationOutput{ID: id, Deleted: true}},
	)
}

func (d *DeleteAnnotation) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (d *DeleteAnnotation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteAnnotation) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteAnnotation) HandleAction(_ core.ActionContext) error {
	return nil
}

func (d *DeleteAnnotation) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteAnnotation) Cleanup(_ core.SetupContext) error {
	return nil
}

func decodeDeleteAnnotationSpec(config any) (DeleteAnnotationSpec, error) {
	m, ok := config.(map[string]any)
	if !ok {
		return DeleteAnnotationSpec{}, fmt.Errorf("error decoding configuration: expected map")
	}
	raw, ok := m["annotationId"]
	if !ok || raw == nil {
		return DeleteAnnotationSpec{}, nil
	}
	return DeleteAnnotationSpec{AnnotationID: normalizeAnnotationIDRaw(raw)}, nil
}

func validateDeleteAnnotationSpec(spec DeleteAnnotationSpec) error {
	s := strings.TrimSpace(spec.AnnotationID)
	if s == "" {
		return errors.New("annotationId is required")
	}
	if isExpressionValue(s) {
		return nil
	}
	_, err := parseAnnotationIDString(s)
	return err
}

// isExpressionValue matches workflow expression syntax (see IntegrationResourceFieldRenderer).
func isExpressionValue(value string) bool {
	value = strings.TrimSpace(value)
	return strings.Contains(value, "{{") || strings.Contains(value, "$[")
}

// normalizeAnnotationIDRaw converts UI / legacy config values to a decimal string ID.
func normalizeAnnotationIDRaw(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func parseAnnotationIDForExecute(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("annotationId is required")
	}
	if isExpressionValue(s) {
		return 0, errors.New("annotationId must resolve to a numeric id before execution")
	}
	return parseAnnotationIDString(s)
}

func parseAnnotationIDString(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("annotationId is required")
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("annotationId must be a positive integer")
	}
	return id, nil
}
