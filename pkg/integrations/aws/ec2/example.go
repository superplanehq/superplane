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

//go:embed example_output_copy_image.json
var exampleOutputCopyImageBytes []byte

//go:embed example_output_deregister_image.json
var exampleOutputDeregisterImageBytes []byte

//go:embed example_output_enable_image.json
var exampleOutputEnableImageBytes []byte

//go:embed example_output_disable_image.json
var exampleOutputDisableImageBytes []byte

//go:embed example_output_enable_image_deprecation.json
var exampleOutputEnableImageDeprecationBytes []byte

//go:embed example_output_disable_image_deprecation.json
var exampleOutputDisableImageDeprecationBytes []byte

var exampleDataOnImageOnce sync.Once
var exampleDataOnImage map[string]any

var exampleOutputCreateImageOnce sync.Once
var exampleOutputCreateImage map[string]any

var exampleOutputGetImageOnce sync.Once
var exampleOutputGetImage map[string]any

var exampleOutputCopyImageOnce sync.Once
var exampleOutputCopyImage map[string]any

var exampleOutputDeregisterImageOnce sync.Once
var exampleOutputDeregisterImage map[string]any

var exampleOutputEnableImageOnce sync.Once
var exampleOutputEnableImage map[string]any

var exampleOutputDisableImageOnce sync.Once
var exampleOutputDisableImage map[string]any

var exampleOutputEnableImageDeprecationOnce sync.Once
var exampleOutputEnableImageDeprecation map[string]any

var exampleOutputDisableImageDeprecationOnce sync.Once
var exampleOutputDisableImageDeprecation map[string]any

func (t *OnImage) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImageOnce, exampleDataOnImageBytes, &exampleDataOnImage)
}

func (c *CreateImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateImageOnce, exampleOutputCreateImageBytes, &exampleOutputCreateImage)
}

func (c *GetImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageOnce, exampleOutputGetImageBytes, &exampleOutputGetImage)
}

func (c *CopyImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCopyImageOnce, exampleOutputCopyImageBytes, &exampleOutputCopyImage)
}

func (c *DeregisterImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeregisterImageOnce,
		exampleOutputDeregisterImageBytes,
		&exampleOutputDeregisterImage,
	)
}

func (c *EnableImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputEnableImageOnce, exampleOutputEnableImageBytes, &exampleOutputEnableImage)
}

func (c *DisableImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDisableImageOnce, exampleOutputDisableImageBytes, &exampleOutputDisableImage)
}

func (c *EnableImageDeprecation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputEnableImageDeprecationOnce,
		exampleOutputEnableImageDeprecationBytes,
		&exampleOutputEnableImageDeprecation,
	)
}

func (c *DisableImageDeprecation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDisableImageDeprecationOnce,
		exampleOutputDisableImageDeprecationBytes,
		&exampleOutputDisableImageDeprecation,
	)
}
