package elastic

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_index_document.json
var exampleOutputIndexDocumentBytes []byte

var exampleOutputIndexDocumentOnce sync.Once
var exampleOutputIndexDocument map[string]any

func (c *IndexDocument) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputIndexDocumentOnce, exampleOutputIndexDocumentBytes, &exampleOutputIndexDocument)
}

//go:embed example_data_on_alert.json
var exampleDataOnAlertBytes []byte

var exampleDataOnAlertOnce sync.Once
var exampleDataOnAlert map[string]any

func (t *OnAlertFires) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertOnce, exampleDataOnAlertBytes, &exampleDataOnAlert)
}
