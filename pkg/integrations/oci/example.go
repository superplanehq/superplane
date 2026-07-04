package oci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_compute_instance.json
var exampleOutputCreateComputeInstanceBytes []byte

//go:embed example_output_create_image.json
var exampleOutputCreateImageBytes []byte

//go:embed example_output_get_image.json
var exampleOutputGetImageBytes []byte

//go:embed example_output_update_image.json
var exampleOutputUpdateImageBytes []byte

//go:embed example_output_delete_image.json
var exampleOutputDeleteImageBytes []byte

var exampleOutputCreateComputeInstanceOnce sync.Once
var exampleOutputCreateComputeInstanceCache map[string]any
var exampleOutputCreateImageOnce sync.Once
var exampleOutputCreateImageCache map[string]any
var exampleOutputGetImageOnce sync.Once
var exampleOutputGetImageCache map[string]any
var exampleOutputUpdateImageOnce sync.Once
var exampleOutputUpdateImageCache map[string]any
var exampleOutputDeleteImageOnce sync.Once
var exampleOutputDeleteImageCache map[string]any

func exampleOutputCreateComputeInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateComputeInstanceOnce, exampleOutputCreateComputeInstanceBytes, &exampleOutputCreateComputeInstanceCache)
}

func exampleOutputCreateImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateImageOnce, exampleOutputCreateImageBytes, &exampleOutputCreateImageCache)
}

func exampleOutputGetImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageOnce, exampleOutputGetImageBytes, &exampleOutputGetImageCache)
}

func exampleOutputUpdateImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateImageOnce, exampleOutputUpdateImageBytes, &exampleOutputUpdateImageCache)
}

func exampleOutputDeleteImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteImageOnce, exampleOutputDeleteImageBytes, &exampleOutputDeleteImageCache)
}

//go:embed example_data_on_compute_instance_created.json
var exampleDataOnComputeInstanceCreatedBytes []byte

var exampleDataOnComputeInstanceCreatedOnce sync.Once
var exampleDataOnComputeInstanceCreatedCache map[string]any

func exampleDataOnComputeInstanceCreated() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnComputeInstanceCreatedOnce, exampleDataOnComputeInstanceCreatedBytes, &exampleDataOnComputeInstanceCreatedCache)
}

//go:embed example_output_create_application.json
var exampleOutputCreateApplicationBytes []byte

var exampleOutputCreateApplicationOnce sync.Once
var exampleOutputCreateApplicationCache map[string]any

func exampleOutputCreateApplication() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateApplicationOnce, exampleOutputCreateApplicationBytes, &exampleOutputCreateApplicationCache)
}

//go:embed example_output_create_function.json
var exampleOutputCreateFunctionBytes []byte

var exampleOutputCreateFunctionOnce sync.Once
var exampleOutputCreateFunctionCache map[string]any

func exampleOutputCreateFunction() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateFunctionOnce, exampleOutputCreateFunctionBytes, &exampleOutputCreateFunctionCache)
}

//go:embed example_output_invoke_function.json
var exampleOutputInvokeFunctionBytes []byte

var exampleOutputInvokeFunctionOnce sync.Once
var exampleOutputInvokeFunctionCache map[string]any

func exampleOutputInvokeFunction() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputInvokeFunctionOnce, exampleOutputInvokeFunctionBytes, &exampleOutputInvokeFunctionCache)
}

//go:embed example_output_delete_application.json
var exampleOutputDeleteApplicationBytes []byte

var exampleOutputDeleteApplicationOnce sync.Once
var exampleOutputDeleteApplicationCache map[string]any

func exampleOutputDeleteApplication() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteApplicationOnce, exampleOutputDeleteApplicationBytes, &exampleOutputDeleteApplicationCache)
}

//go:embed example_output_delete_function.json
var exampleOutputDeleteFunctionBytes []byte

var exampleOutputDeleteFunctionOnce sync.Once
var exampleOutputDeleteFunctionCache map[string]any

func exampleOutputDeleteFunction() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteFunctionOnce, exampleOutputDeleteFunctionBytes, &exampleOutputDeleteFunctionCache)
}

//go:embed example_output_get_instance.json
var exampleOutputGetInstanceBytes []byte

var exampleOutputGetInstanceOnce sync.Once
var exampleOutputGetInstanceCache map[string]any

func exampleOutputGetInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetInstanceOnce, exampleOutputGetInstanceBytes, &exampleOutputGetInstanceCache)
}

//go:embed example_output_update_instance.json
var exampleOutputUpdateInstanceBytes []byte

var exampleOutputUpdateInstanceOnce sync.Once
var exampleOutputUpdateInstanceCache map[string]any

func exampleOutputUpdateInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateInstanceOnce, exampleOutputUpdateInstanceBytes, &exampleOutputUpdateInstanceCache)
}

//go:embed example_output_manage_instance_power.json
var exampleOutputManageInstancePowerBytes []byte

var exampleOutputManageInstancePowerOnce sync.Once
var exampleOutputManageInstancePowerCache map[string]any

func exampleOutputManageInstancePower() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputManageInstancePowerOnce, exampleOutputManageInstancePowerBytes, &exampleOutputManageInstancePowerCache)
}

//go:embed example_output_delete_instance.json
var exampleOutputDeleteInstanceBytes []byte

var exampleOutputDeleteInstanceOnce sync.Once
var exampleOutputDeleteInstanceCache map[string]any

func exampleOutputDeleteInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteInstanceOnce, exampleOutputDeleteInstanceBytes, &exampleOutputDeleteInstanceCache)
}

//go:embed example_data_on_instance_state_change.json
var exampleDataOnInstanceStateChangeBytes []byte

var exampleDataOnInstanceStateChangeOnce sync.Once
var exampleDataOnInstanceStateChangeCache map[string]any

func exampleDataOnInstanceStateChange() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnInstanceStateChangeOnce, exampleDataOnInstanceStateChangeBytes, &exampleDataOnInstanceStateChangeCache)
}
