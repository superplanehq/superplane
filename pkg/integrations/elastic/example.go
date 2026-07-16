package elastic

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_index_document.json
var exampleOutputIndexDocumentBytes []byte
var exampleOutputIndexDocument = utils.NewEmbeddedJSON(exampleOutputIndexDocumentBytes)

func (c *IndexDocument) ExampleOutput() map[string]any {
	return exampleOutputIndexDocument.Value()
}

//go:embed example_output_create_case.json
var exampleOutputCreateCaseBytes []byte
var exampleOutputCreateCase = utils.NewEmbeddedJSON(exampleOutputCreateCaseBytes)

func (c *CreateCase) ExampleOutput() map[string]any {
	return exampleOutputCreateCase.Value()
}

//go:embed example_output_get_case.json
var exampleOutputGetCaseBytes []byte
var exampleOutputGetCase = utils.NewEmbeddedJSON(exampleOutputGetCaseBytes)

func (c *GetCase) ExampleOutput() map[string]any {
	return exampleOutputGetCase.Value()
}

//go:embed example_output_update_case.json
var exampleOutputUpdateCaseBytes []byte
var exampleOutputUpdateCase = utils.NewEmbeddedJSON(exampleOutputUpdateCaseBytes)

func (c *UpdateCase) ExampleOutput() map[string]any {
	return exampleOutputUpdateCase.Value()
}

//go:embed example_output_get_document.json
var exampleOutputGetDocumentBytes []byte
var exampleOutputGetDocument = utils.NewEmbeddedJSON(exampleOutputGetDocumentBytes)

func (c *GetDocument) ExampleOutput() map[string]any {
	return exampleOutputGetDocument.Value()
}

//go:embed example_output_update_document.json
var exampleOutputUpdateDocumentBytes []byte
var exampleOutputUpdateDocument = utils.NewEmbeddedJSON(exampleOutputUpdateDocumentBytes)

func (c *UpdateDocument) ExampleOutput() map[string]any {
	return exampleOutputUpdateDocument.Value()
}

//go:embed example_data_on_alert.json
var exampleDataOnAlertBytes []byte
var exampleDataOnAlert = utils.NewEmbeddedJSON(exampleDataOnAlertBytes)

func (t *OnAlertFires) ExampleData() map[string]any {
	return exampleDataOnAlert.Value()
}

//go:embed example_data_on_case_status_change.json
var exampleDataOnCaseStatusChangeBytes []byte
var exampleDataOnCaseStatusChange = utils.NewEmbeddedJSON(exampleDataOnCaseStatusChangeBytes)

func (t *OnCaseStatusChange) ExampleData() map[string]any {
	return exampleDataOnCaseStatusChange.Value()
}

//go:embed example_data_on_document_indexed.json
var exampleDataOnDocumentIndexedBytes []byte
var exampleDataOnDocumentIndexed = utils.NewEmbeddedJSON(exampleDataOnDocumentIndexedBytes)

func (t *OnDocumentIndexed) ExampleData() map[string]any {
	return exampleDataOnDocumentIndexed.Value()
}
