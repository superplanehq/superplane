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

//go:embed example_output_create_case.json
var exampleOutputCreateCaseBytes []byte

var exampleOutputCreateCaseOnce sync.Once
var exampleOutputCreateCase map[string]any

func (c *CreateCase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateCaseOnce, exampleOutputCreateCaseBytes, &exampleOutputCreateCase)
}

//go:embed example_output_get_case.json
var exampleOutputGetCaseBytes []byte

var exampleOutputGetCaseOnce sync.Once
var exampleOutputGetCase map[string]any

func (c *GetCase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCaseOnce, exampleOutputGetCaseBytes, &exampleOutputGetCase)
}

//go:embed example_output_update_case.json
var exampleOutputUpdateCaseBytes []byte

var exampleOutputUpdateCaseOnce sync.Once
var exampleOutputUpdateCase map[string]any

func (c *UpdateCase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateCaseOnce, exampleOutputUpdateCaseBytes, &exampleOutputUpdateCase)
}

//go:embed example_output_get_document.json
var exampleOutputGetDocumentBytes []byte

var exampleOutputGetDocumentOnce sync.Once
var exampleOutputGetDocument map[string]any

func (c *GetDocument) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetDocumentOnce, exampleOutputGetDocumentBytes, &exampleOutputGetDocument)
}

//go:embed example_output_update_document.json
var exampleOutputUpdateDocumentBytes []byte

var exampleOutputUpdateDocumentOnce sync.Once
var exampleOutputUpdateDocument map[string]any

func (c *UpdateDocument) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateDocumentOnce, exampleOutputUpdateDocumentBytes, &exampleOutputUpdateDocument)
}

//go:embed example_data_on_alert.json
var exampleDataOnAlertBytes []byte

var exampleDataOnAlertOnce sync.Once
var exampleDataOnAlert map[string]any

func (t *OnAlertFires) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertOnce, exampleDataOnAlertBytes, &exampleDataOnAlert)
}

//go:embed example_data_on_case_status_change.json
var exampleDataOnCaseStatusChangeBytes []byte

var exampleDataOnCaseStatusChangeOnce sync.Once
var exampleDataOnCaseStatusChange map[string]any

func (t *OnCaseStatusChange) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnCaseStatusChangeOnce, exampleDataOnCaseStatusChangeBytes, &exampleDataOnCaseStatusChange)
}

//go:embed example_data_on_document_indexed.json
var exampleDataOnDocumentIndexedBytes []byte

var exampleDataOnDocumentIndexedOnce sync.Once
var exampleDataOnDocumentIndexed map[string]any

func (t *OnDocumentIndexed) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnDocumentIndexedOnce, exampleDataOnDocumentIndexedBytes, &exampleDataOnDocumentIndexed)
}
