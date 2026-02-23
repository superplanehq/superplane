package prometheus

import (
	_ "embed"
	"sync"

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

var exampleDataOnAlertOnce sync.Once
var exampleDataOnAlert map[string]any

var exampleOutputGetAlertOnce sync.Once
var exampleOutputGetAlert map[string]any

var exampleOutputCreateSilenceOnce sync.Once
var exampleOutputCreateSilence map[string]any

var exampleOutputExpireSilenceOnce sync.Once
var exampleOutputExpireSilence map[string]any

var exampleOutputGetSilenceOnce sync.Once
var exampleOutputGetSilence map[string]any

var exampleOutputQueryOnce sync.Once
var exampleOutputQuery map[string]any

var exampleOutputQueryRangeOnce sync.Once
var exampleOutputQueryRange map[string]any

func (t *OnAlert) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertOnce, exampleDataOnAlertBytes, &exampleDataOnAlert)
}

func (c *GetAlert) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetAlertOnce, exampleOutputGetAlertBytes, &exampleOutputGetAlert)
}

func (c *CreateSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSilenceOnce, exampleOutputCreateSilenceBytes, &exampleOutputCreateSilence)
}

func (c *ExpireSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputExpireSilenceOnce, exampleOutputExpireSilenceBytes, &exampleOutputExpireSilence)
}

func (c *GetSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSilenceOnce, exampleOutputGetSilenceBytes, &exampleOutputGetSilence)
}

func (c *Query) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryOnce, exampleOutputQueryBytes, &exampleOutputQuery)
}

func (c *QueryRange) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryRangeOnce, exampleOutputQueryRangeBytes, &exampleOutputQueryRange)
}
