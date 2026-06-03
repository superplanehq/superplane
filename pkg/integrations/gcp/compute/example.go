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

//go:embed example_output_manage_vm_instance_power.json
var exampleOutputManageVMInstancePowerBytes []byte

//go:embed example_output_update_vm_instance_type.json
var exampleOutputUpdateVMInstanceTypeBytes []byte

//go:embed example_output_get_vm_instance_metrics.json
var exampleOutputGetVMInstanceMetricsBytes []byte

//go:embed example_output_create_image.json
var exampleOutputCreateImageBytes []byte

//go:embed example_output_update_image.json
var exampleOutputUpdateImageBytes []byte

//go:embed example_output_delete_image.json
var exampleOutputDeleteImageBytes []byte

var (
	exampleOutputCreateVMOnce sync.Once
	exampleOutputCreateVM     map[string]any

	exampleOutputDeleteVMInstanceOnce sync.Once
	exampleOutputDeleteVMInstance     map[string]any

	exampleDataOnVMInstanceOnce sync.Once
	exampleDataOnVMInstance     map[string]any

	exampleOutputManageVMInstancePowerOnce sync.Once
	exampleOutputManageVMInstancePower     map[string]any

	exampleOutputUpdateVMInstanceTypeOnce sync.Once
	exampleOutputUpdateVMInstanceType     map[string]any

	exampleOutputGetVMInstanceMetricsOnce sync.Once
	exampleOutputGetVMInstanceMetrics     map[string]any

	exampleOutputCreateImageOnce sync.Once
	exampleOutputCreateImage     map[string]any

	exampleOutputUpdateImageOnce sync.Once
	exampleOutputUpdateImage     map[string]any

	exampleOutputDeleteImageOnce sync.Once
	exampleOutputDeleteImage     map[string]any
)

func (c *CreateVM) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateVMOnce, exampleOutputCreateVMBytes, &exampleOutputCreateVM)
}

func (d *DeleteVMInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteVMInstanceOnce, exampleOutputDeleteVMInstanceBytes, &exampleOutputDeleteVMInstance)
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

func (c *CreateImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateImageOnce, exampleOutputCreateImageBytes, &exampleOutputCreateImage)
}

func (u *UpdateImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateImageOnce, exampleOutputUpdateImageBytes, &exampleOutputUpdateImage)
}

func (d *DeleteImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteImageOnce, exampleOutputDeleteImageBytes, &exampleOutputDeleteImage)
}
