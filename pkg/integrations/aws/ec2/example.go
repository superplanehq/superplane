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

//go:embed example_output_get_instance_status.json
var exampleOutputGetInstanceStatusBytes []byte

//go:embed example_output_start_instance.json
var exampleOutputStartInstanceBytes []byte

//go:embed example_output_stop_instance.json
var exampleOutputStopInstanceBytes []byte

//go:embed example_output_reboot_instance.json
var exampleOutputRebootInstanceBytes []byte

//go:embed example_output_run_instance.json
var exampleOutputRunInstanceBytes []byte

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

var exampleOutputGetInstanceStatusOnce sync.Once
var exampleOutputGetInstanceStatus map[string]any

var exampleOutputStartInstanceOnce sync.Once
var exampleOutputStartInstance map[string]any

var exampleOutputStopInstanceOnce sync.Once
var exampleOutputStopInstance map[string]any

var exampleOutputRebootInstanceOnce sync.Once
var exampleOutputRebootInstance map[string]any

var exampleOutputRunInstanceOnce sync.Once
var exampleOutputRunInstance map[string]any

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

func (c *GetInstanceStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetInstanceStatusOnce, exampleOutputGetInstanceStatusBytes, &exampleOutputGetInstanceStatus)
}

func (c *StartInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputStartInstanceOnce, exampleOutputStartInstanceBytes, &exampleOutputStartInstance)
}

func (c *StopInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputStopInstanceOnce, exampleOutputStopInstanceBytes, &exampleOutputStopInstance)
}

func (c *RebootInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRebootInstanceOnce, exampleOutputRebootInstanceBytes, &exampleOutputRebootInstance)
}

func (c *RunInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunInstanceOnce, exampleOutputRunInstanceBytes, &exampleOutputRunInstance)
}
