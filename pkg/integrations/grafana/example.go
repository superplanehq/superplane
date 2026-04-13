package grafana

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_data_source.json
var exampleOutputQueryDataSourceBytes []byte

//go:embed example_data_on_alert_firing.json
var exampleDataOnAlertFiringBytes []byte

//go:embed example_output_create_annotation.json
var exampleOutputCreateAnnotationBytes []byte

//go:embed example_output_list_annotations.json
var exampleOutputListAnnotationsBytes []byte

//go:embed example_output_delete_annotation.json
var exampleOutputDeleteAnnotationBytes []byte

var exampleOutputQueryDataSourceOnce sync.Once
var exampleOutputQueryDataSource map[string]any

var exampleDataOnAlertFiringOnce sync.Once
var exampleDataOnAlertFiring map[string]any

var exampleOutputCreateAnnotationOnce sync.Once
var exampleOutputCreateAnnotation map[string]any

var exampleOutputListAnnotationsOnce sync.Once
var exampleOutputListAnnotations map[string]any

var exampleOutputDeleteAnnotationOnce sync.Once
var exampleOutputDeleteAnnotation map[string]any

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryDataSourceOnce, exampleOutputQueryDataSourceBytes, &exampleOutputQueryDataSource)
}

func (t *OnAlertFiring) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertFiringOnce, exampleDataOnAlertFiringBytes, &exampleDataOnAlertFiring)
}

func (c *CreateAnnotation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateAnnotationOnce, exampleOutputCreateAnnotationBytes, &exampleOutputCreateAnnotation)
}

func (l *ListAnnotations) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListAnnotationsOnce, exampleOutputListAnnotationsBytes, &exampleOutputListAnnotations)
}

func (d *DeleteAnnotation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteAnnotationOnce, exampleOutputDeleteAnnotationBytes, &exampleOutputDeleteAnnotation)
}
