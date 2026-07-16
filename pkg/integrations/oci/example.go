package oci

import (
	_ "embed"

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
var exampleOutputCreateComputeInstanceCache = utils.NewEmbeddedJSON(exampleOutputCreateComputeInstanceBytes)
var exampleOutputCreateImageCache = utils.NewEmbeddedJSON(exampleOutputCreateImageBytes)
var exampleOutputGetImageCache = utils.NewEmbeddedJSON(exampleOutputGetImageBytes)
var exampleOutputUpdateImageCache = utils.NewEmbeddedJSON(exampleOutputUpdateImageBytes)
var exampleOutputDeleteImageCache = utils.NewEmbeddedJSON(exampleOutputDeleteImageBytes)

func exampleOutputCreateComputeInstance() map[string]any {
	return exampleOutputCreateComputeInstanceCache.Value()
}

func exampleOutputCreateImage() map[string]any {
	return exampleOutputCreateImageCache.Value()
}

func exampleOutputGetImage() map[string]any {
	return exampleOutputGetImageCache.Value()
}

func exampleOutputUpdateImage() map[string]any {
	return exampleOutputUpdateImageCache.Value()
}

func exampleOutputDeleteImage() map[string]any {
	return exampleOutputDeleteImageCache.Value()
}

//go:embed example_data_on_compute_instance_created.json
var exampleDataOnComputeInstanceCreatedBytes []byte
var exampleDataOnComputeInstanceCreatedCache = utils.NewEmbeddedJSON(exampleDataOnComputeInstanceCreatedBytes)

func exampleDataOnComputeInstanceCreated() map[string]any {
	return exampleDataOnComputeInstanceCreatedCache.Value()
}

//go:embed example_output_create_application.json
var exampleOutputCreateApplicationBytes []byte
var exampleOutputCreateApplicationCache = utils.NewEmbeddedJSON(exampleOutputCreateApplicationBytes)

func exampleOutputCreateApplication() map[string]any {
	return exampleOutputCreateApplicationCache.Value()
}

//go:embed example_output_create_function.json
var exampleOutputCreateFunctionBytes []byte
var exampleOutputCreateFunctionCache = utils.NewEmbeddedJSON(exampleOutputCreateFunctionBytes)

func exampleOutputCreateFunction() map[string]any {
	return exampleOutputCreateFunctionCache.Value()
}

//go:embed example_output_invoke_function.json
var exampleOutputInvokeFunctionBytes []byte
var exampleOutputInvokeFunctionCache = utils.NewEmbeddedJSON(exampleOutputInvokeFunctionBytes)

func exampleOutputInvokeFunction() map[string]any {
	return exampleOutputInvokeFunctionCache.Value()
}

//go:embed example_output_delete_application.json
var exampleOutputDeleteApplicationBytes []byte
var exampleOutputDeleteApplicationCache = utils.NewEmbeddedJSON(exampleOutputDeleteApplicationBytes)

func exampleOutputDeleteApplication() map[string]any {
	return exampleOutputDeleteApplicationCache.Value()
}

//go:embed example_output_delete_function.json
var exampleOutputDeleteFunctionBytes []byte
var exampleOutputDeleteFunctionCache = utils.NewEmbeddedJSON(exampleOutputDeleteFunctionBytes)

func exampleOutputDeleteFunction() map[string]any {
	return exampleOutputDeleteFunctionCache.Value()
}

//go:embed example_output_get_instance.json
var exampleOutputGetInstanceBytes []byte
var exampleOutputGetInstanceCache = utils.NewEmbeddedJSON(exampleOutputGetInstanceBytes)

func exampleOutputGetInstance() map[string]any {
	return exampleOutputGetInstanceCache.Value()
}

//go:embed example_output_update_instance.json
var exampleOutputUpdateInstanceBytes []byte
var exampleOutputUpdateInstanceCache = utils.NewEmbeddedJSON(exampleOutputUpdateInstanceBytes)

func exampleOutputUpdateInstance() map[string]any {
	return exampleOutputUpdateInstanceCache.Value()
}

//go:embed example_output_manage_instance_power.json
var exampleOutputManageInstancePowerBytes []byte
var exampleOutputManageInstancePowerCache = utils.NewEmbeddedJSON(exampleOutputManageInstancePowerBytes)

func exampleOutputManageInstancePower() map[string]any {
	return exampleOutputManageInstancePowerCache.Value()
}

//go:embed example_output_delete_instance.json
var exampleOutputDeleteInstanceBytes []byte
var exampleOutputDeleteInstanceCache = utils.NewEmbeddedJSON(exampleOutputDeleteInstanceBytes)

func exampleOutputDeleteInstance() map[string]any {
	return exampleOutputDeleteInstanceCache.Value()
}

//go:embed example_data_on_instance_state_change.json
var exampleDataOnInstanceStateChangeBytes []byte
var exampleDataOnInstanceStateChangeCache = utils.NewEmbeddedJSON(exampleDataOnInstanceStateChangeBytes)

func exampleDataOnInstanceStateChange() map[string]any {
	return exampleDataOnInstanceStateChangeCache.Value()
}
