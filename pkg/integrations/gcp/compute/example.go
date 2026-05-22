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

var (
	exampleOutputCreateVMOnce sync.Once
	exampleOutputCreateVM     map[string]any

	exampleOutputDeleteVMInstanceOnce sync.Once
	exampleOutputDeleteVMInstance     map[string]any

	exampleDataOnVMInstanceOnce sync.Once
	exampleDataOnVMInstance     map[string]any

	exampleOutputGetVMInstanceOnce sync.Once
	exampleOutputGetVMInstance     map[string]any
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
