package ec2

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_image.json
var exampleDataOnImageBytes []byte

//go:embed example_output_create_image.json
var exampleOutputCreateImageBytes []byte

//go:embed example_output_get_image.json
var exampleOutputGetImageBytes []byte

var exampleDataOnImageOnce sync.Once
var exampleDataOnImage map[string]any

var exampleOutputCreateImageOnce sync.Once
var exampleOutputCreateImage map[string]any

var exampleOutputGetImageOnce sync.Once
var exampleOutputGetImage map[string]any

func (t *OnImage) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImageOnce, exampleDataOnImageBytes, &exampleDataOnImage)
}

func (c *CreateImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateImageOnce, exampleOutputCreateImageBytes, &exampleOutputCreateImage)
}

func (c *GetImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageOnce, exampleOutputGetImageBytes, &exampleOutputGetImage)
}
