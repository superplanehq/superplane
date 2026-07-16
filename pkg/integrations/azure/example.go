package azure

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_vm.json
var exampleOutputCreateVMBytes []byte

//go:embed example_output_start_vm.json
var exampleOutputStartVMBytes []byte

//go:embed example_output_stop_vm.json
var exampleOutputStopVMBytes []byte

//go:embed example_output_restart_vm.json
var exampleOutputRestartVMBytes []byte

//go:embed example_output_deallocate_vm.json
var exampleOutputDeallocateVMBytes []byte

//go:embed example_output_delete_vm.json
var exampleOutputDeleteVMBytes []byte

//go:embed example_data_on_blob_created.json
var exampleDataOnBlobCreatedBytes []byte

//go:embed example_data_on_blob_deleted.json
var exampleDataOnBlobDeletedBytes []byte

//go:embed example_data_on_image_pushed.json
var exampleDataOnImagePushedBytes []byte

//go:embed example_data_on_image_deleted.json
var exampleDataOnImageDeletedBytes []byte

//go:embed example_data_on_vm_started.json
var exampleDataOnVMStartedBytes []byte

//go:embed example_data_on_vm_stopped.json
var exampleDataOnVMStoppedBytes []byte

//go:embed example_data_on_vm_restarted.json
var exampleDataOnVMRestartedBytes []byte

//go:embed example_data_on_vm_deallocated.json
var exampleDataOnVMDeallocatedBytes []byte

//go:embed example_data_on_vm_deleted.json
var exampleDataOnVMDeletedBytes []byte

var (
	exampleOutputCreateVM      = utils.NewEmbeddedJSON(exampleOutputCreateVMBytes)
	exampleOutputStartVM       = utils.NewEmbeddedJSON(exampleOutputStartVMBytes)
	exampleOutputStopVM        = utils.NewEmbeddedJSON(exampleOutputStopVMBytes)
	exampleOutputRestartVM     = utils.NewEmbeddedJSON(exampleOutputRestartVMBytes)
	exampleOutputDeallocateVM  = utils.NewEmbeddedJSON(exampleOutputDeallocateVMBytes)
	exampleOutputDeleteVM      = utils.NewEmbeddedJSON(exampleOutputDeleteVMBytes)
	exampleDataOnBlobCreated   = utils.NewEmbeddedJSON(exampleDataOnBlobCreatedBytes)
	exampleDataOnBlobDeleted   = utils.NewEmbeddedJSON(exampleDataOnBlobDeletedBytes)
	exampleDataOnImagePushed   = utils.NewEmbeddedJSON(exampleDataOnImagePushedBytes)
	exampleDataOnImageDeleted  = utils.NewEmbeddedJSON(exampleDataOnImageDeletedBytes)
	exampleDataOnVMStarted     = utils.NewEmbeddedJSON(exampleDataOnVMStartedBytes)
	exampleDataOnVMStopped     = utils.NewEmbeddedJSON(exampleDataOnVMStoppedBytes)
	exampleDataOnVMRestarted   = utils.NewEmbeddedJSON(exampleDataOnVMRestartedBytes)
	exampleDataOnVMDeallocated = utils.NewEmbeddedJSON(exampleDataOnVMDeallocatedBytes)
	exampleDataOnVMDeleted     = utils.NewEmbeddedJSON(exampleDataOnVMDeletedBytes)
)

func (c *CreateVMComponent) ExampleOutput() map[string]any {
	return exampleOutputCreateVM.Value()
}

func (c *StartVMComponent) ExampleOutput() map[string]any {
	return exampleOutputStartVM.Value()
}

func (c *StopVMComponent) ExampleOutput() map[string]any {
	return exampleOutputStopVM.Value()
}

func (c *RestartVMComponent) ExampleOutput() map[string]any {
	return exampleOutputRestartVM.Value()
}

func (c *DeallocateVMComponent) ExampleOutput() map[string]any {
	return exampleOutputDeallocateVM.Value()
}

func (c *DeleteVMComponent) ExampleOutput() map[string]any {
	return exampleOutputDeleteVM.Value()
}

func (t *OnBlobCreated) ExampleData() map[string]any {
	return exampleDataOnBlobCreated.Value()
}

func (t *OnBlobDeleted) ExampleData() map[string]any {
	return exampleDataOnBlobDeleted.Value()
}

func (t *OnImagePushed) ExampleData() map[string]any {
	return exampleDataOnImagePushed.Value()
}

func (t *OnImageDeleted) ExampleData() map[string]any {
	return exampleDataOnImageDeleted.Value()
}

func (t *OnVMStarted) ExampleData() map[string]any {
	return exampleDataOnVMStarted.Value()
}

func (t *OnVMStopped) ExampleData() map[string]any {
	return exampleDataOnVMStopped.Value()
}

func (t *OnVMRestarted) ExampleData() map[string]any {
	return exampleDataOnVMRestarted.Value()
}

func (t *OnVMDeallocated) ExampleData() map[string]any {
	return exampleDataOnVMDeallocated.Value()
}

func (t *OnVMDeleted) ExampleData() map[string]any {
	return exampleDataOnVMDeleted.Value()
}
