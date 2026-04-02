package grafana

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteAnnotation struct{}

type DeleteAnnotationSpec struct {
	AnnotationID int64 `json:"annotationId" mapstructure:"annotationId"`
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
	return `The Delete Annotation component removes an annotation from Grafana by its ID.

## Use Cases

- **Cleanup incorrect markers**: Remove an annotation that was created with wrong text or tags
- **Automated lifecycle**: Delete temporary markers (e.g. maintenance window start) once the event is complete
- **Idempotent workflows**: Allow re-runs to clean up previously created annotations before re-creating them

## Configuration

- **Annotation ID**: The numeric ID of the annotation to delete (required)

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
			Label:       "Annotation ID",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "The numeric ID of the annotation to delete",
		},
	}
}

func (d *DeleteAnnotation) Setup(ctx core.SetupContext) error {
	spec, err := decodeDeleteAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateDeleteAnnotationSpec(spec)
}

func (d *DeleteAnnotation) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeDeleteAnnotationSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeleteAnnotationSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteAnnotation(spec.AnnotationID); err != nil {
		return fmt.Errorf("error deleting annotation: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.annotation.deleted",
		[]any{DeleteAnnotationOutput{ID: spec.AnnotationID, Deleted: true}},
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
	spec := DeleteAnnotationSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return DeleteAnnotationSpec{}, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return DeleteAnnotationSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateDeleteAnnotationSpec(spec DeleteAnnotationSpec) error {
	if spec.AnnotationID <= 0 {
		return errors.New("annotationId is required")
	}
	return nil
}
