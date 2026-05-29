package compute

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_vm.json
var exampleOutputCreateVMBytes []byte

//go:embed example_output_delete_vm_instance.json
var exampleOutputDeleteVMInstanceBytes []byte

//go:embed example_data_on_vm_instance.json
var exampleDataOnVMInstanceBytes []byte

//go:embed example_output_get_vm_instance.json
var exampleOutputGetVMInstanceBytes []byte

//go:embed example_output_manage_vm_instance_power.json
var exampleOutputManageVMInstancePowerBytes []byte

//go:embed example_output_update_vm_instance_type.json
var exampleOutputUpdateVMInstanceTypeBytes []byte

//go:embed example_output_get_vm_instance_metrics.json
var exampleOutputGetVMInstanceMetricsBytes []byte

var (
	exampleOutputCreateVMOnce sync.Once
	exampleOutputCreateVM     map[string]any

	exampleOutputDeleteVMInstanceOnce sync.Once
	exampleOutputDeleteVMInstance     map[string]any

	exampleDataOnVMInstanceOnce sync.Once
	exampleDataOnVMInstance     map[string]any

	exampleOutputGetVMInstanceOnce sync.Once
	exampleOutputGetVMInstance     map[string]any

	exampleOutputManageVMInstancePowerOnce sync.Once
	exampleOutputManageVMInstancePower     map[string]any

	exampleOutputUpdateVMInstanceTypeOnce sync.Once
	exampleOutputUpdateVMInstanceType     map[string]any

	exampleOutputGetVMInstanceMetricsOnce sync.Once
	exampleOutputGetVMInstanceMetrics     map[string]any
)

func (c *CreateVM) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateVMOnce, exampleOutputCreateVMBytes, &exampleOutputCreateVM)
}

func (d *DeleteVMInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteVMInstanceOnce, exampleOutputDeleteVMInstanceBytes, &exampleOutputDeleteVMInstance)
}

func (g *GetVMInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetVMInstanceOnce, exampleOutputGetVMInstanceBytes, &exampleOutputGetVMInstance)
}

func (t *OnVMInstance) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVMInstanceOnce, exampleDataOnVMInstanceBytes, &exampleDataOnVMInstance)
}

func (m *ManageVMInstancePower) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputManageVMInstancePowerOnce, exampleOutputManageVMInstancePowerBytes, &exampleOutputManageVMInstancePower)
}

func (u *UpdateVMInstanceType) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateVMInstanceTypeOnce, exampleOutputUpdateVMInstanceTypeBytes, &exampleOutputUpdateVMInstanceType)
}

func (g *GetVMInstanceMetrics) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetVMInstanceMetricsOnce, exampleOutputGetVMInstanceMetricsBytes, &exampleOutputGetVMInstanceMetrics)
}
