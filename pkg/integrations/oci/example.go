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
