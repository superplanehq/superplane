package oci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_compute_instance.json
var exampleOutputCreateComputeInstanceBytes []byte

var exampleOutputCreateComputeInstanceOnce sync.Once
var exampleOutputCreateComputeInstanceCache map[string]any

func exampleOutputCreateComputeInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateComputeInstanceOnce, exampleOutputCreateComputeInstanceBytes, &exampleOutputCreateComputeInstanceCache)
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
