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

var exampleDataOnAlertOnce sync.Once
var exampleDataOnAlert map[string]any

var exampleOutputGetAlertOnce sync.Once
var exampleOutputGetAlert map[string]any

var exampleOutputCreateSilenceOnce sync.Once
var exampleOutputCreateSilence map[string]any

var exampleOutputExpireSilenceOnce sync.Once
var exampleOutputExpireSilence map[string]any

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
