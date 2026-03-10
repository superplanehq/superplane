package cloudstorage

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_object.json
var exampleOutputGetObjectBytes []byte

var exampleOutputGetObjectOnce sync.Once
var exampleOutputGetObject map[string]any

func (c *GetObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetObjectOnce, exampleOutputGetObjectBytes, &exampleOutputGetObject)
}

//go:embed example_output_upload_object.json
var exampleOutputUploadObjectBytes []byte

var exampleOutputUploadObjectOnce sync.Once
var exampleOutputUploadObject map[string]any

func (c *UploadObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUploadObjectOnce, exampleOutputUploadObjectBytes, &exampleOutputUploadObject)
}
