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

//go:embed example_output_list_silences.json
var exampleOutputListSilencesBytes []byte

//go:embed example_output_get_silence.json
var exampleOutputGetSilenceBytes []byte

//go:embed example_output_create_silence.json
var exampleOutputCreateSilenceBytes []byte

//go:embed example_output_delete_silence.json
var exampleOutputDeleteSilenceBytes []byte

var exampleOutputQueryDataSourceOnce sync.Once
var exampleOutputQueryDataSource map[string]any

var exampleDataOnAlertFiringOnce sync.Once
var exampleDataOnAlertFiring map[string]any

var exampleOutputListSilencesOnce sync.Once
var exampleOutputListSilences map[string]any

var exampleOutputGetSilenceOnce sync.Once
var exampleOutputGetSilence map[string]any

var exampleOutputCreateSilenceOnce sync.Once
var exampleOutputCreateSilence map[string]any

var exampleOutputDeleteSilenceOnce sync.Once
var exampleOutputDeleteSilence map[string]any

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryDataSourceOnce, exampleOutputQueryDataSourceBytes, &exampleOutputQueryDataSource)
}

func (t *OnAlertFiring) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertFiringOnce, exampleDataOnAlertFiringBytes, &exampleDataOnAlertFiring)
}

func (l *ListSilences) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListSilencesOnce, exampleOutputListSilencesBytes, &exampleOutputListSilences)
}

func (g *GetSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSilenceOnce, exampleOutputGetSilenceBytes, &exampleOutputGetSilence)
}

func (c *CreateSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSilenceOnce, exampleOutputCreateSilenceBytes, &exampleOutputCreateSilence)
}

func (d *DeleteSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteSilenceOnce, exampleOutputDeleteSilenceBytes, &exampleOutputDeleteSilence)
}
