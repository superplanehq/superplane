package prometheus

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alert.json
var exampleDataOnAlertBytes []byte

//go:embed example_output_get_alert.json
var exampleOutputGetAlertBytes []byte

//go:embed example_output_create_silence.json
var exampleOutputCreateSilenceBytes []byte

//go:embed example_output_expire_silence.json
var exampleOutputExpireSilenceBytes []byte

//go:embed example_output_get_silence.json
var exampleOutputGetSilenceBytes []byte

//go:embed example_output_query.json
var exampleOutputQueryBytes []byte

//go:embed example_output_query_range.json
var exampleOutputQueryRangeBytes []byte
var exampleDataOnAlert = utils.NewEmbeddedJSON(exampleDataOnAlertBytes)
var exampleOutputGetAlert = utils.NewEmbeddedJSON(exampleOutputGetAlertBytes)
var exampleOutputCreateSilence = utils.NewEmbeddedJSON(exampleOutputCreateSilenceBytes)
var exampleOutputExpireSilence = utils.NewEmbeddedJSON(exampleOutputExpireSilenceBytes)
var exampleOutputGetSilence = utils.NewEmbeddedJSON(exampleOutputGetSilenceBytes)
var exampleOutputQuery = utils.NewEmbeddedJSON(exampleOutputQueryBytes)
var exampleOutputQueryRange = utils.NewEmbeddedJSON(exampleOutputQueryRangeBytes)

func (t *OnAlert) ExampleData() map[string]any {
	return exampleDataOnAlert.Value()
}

func (c *GetAlert) ExampleOutput() map[string]any {
	return exampleOutputGetAlert.Value()
}

func (c *CreateSilence) ExampleOutput() map[string]any {
	return exampleOutputCreateSilence.Value()
}

func (c *ExpireSilence) ExampleOutput() map[string]any {
	return exampleOutputExpireSilence.Value()
}

func (c *GetSilence) ExampleOutput() map[string]any {
	return exampleOutputGetSilence.Value()
}

func (c *Query) ExampleOutput() map[string]any {
	return exampleOutputQuery.Value()
}

func (c *QueryRange) ExampleOutput() map[string]any {
	return exampleOutputQueryRange.Value()
}
