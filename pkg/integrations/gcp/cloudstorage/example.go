package cloudstorage

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_object.json
var exampleOutputGetObjectBytes []byte

//go:embed example_output_upload_object.json
var exampleOutputUploadObjectBytes []byte

//go:embed example_data_on_object_finalized.json
var exampleDataOnObjectFinalizedBytes []byte

var exampleOutputGetObjectOnce sync.Once
var exampleOutputGetObject map[string]any

var exampleOutputUploadObjectOnce sync.Once
var exampleOutputUploadObject map[string]any

var exampleDataOnObjectFinalizedOnce sync.Once
var exampleDataOnObjectFinalized map[string]any

func (c *GetObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetObjectOnce, exampleOutputGetObjectBytes, &exampleOutputGetObject)
}

func (c *UploadObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUploadObjectOnce, exampleOutputUploadObjectBytes, &exampleOutputUploadObject)
}

func (t *OnObjectFinalized) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnObjectFinalizedOnce, exampleDataOnObjectFinalizedBytes, &exampleDataOnObjectFinalized)
}
