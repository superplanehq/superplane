package azure

import (
	_ "embed"
	"sync"

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
	exampleOutputCreateVMOnce sync.Once
	exampleOutputCreateVM     map[string]any

	exampleOutputStartVMOnce sync.Once
	exampleOutputStartVM     map[string]any

	exampleOutputStopVMOnce sync.Once
	exampleOutputStopVM     map[string]any

	exampleOutputRestartVMOnce sync.Once
	exampleOutputRestartVM     map[string]any

	exampleOutputDeallocateVMOnce sync.Once
	exampleOutputDeallocateVM     map[string]any

	exampleOutputDeleteVMOnce sync.Once
	exampleOutputDeleteVM     map[string]any

	exampleDataOnBlobCreatedOnce sync.Once
	exampleDataOnBlobCreated     map[string]any

	exampleDataOnBlobDeletedOnce sync.Once
	exampleDataOnBlobDeleted     map[string]any

	exampleDataOnImagePushedOnce sync.Once
	exampleDataOnImagePushed     map[string]any

	exampleDataOnImageDeletedOnce sync.Once
	exampleDataOnImageDeleted     map[string]any

	exampleDataOnVMStartedOnce sync.Once
	exampleDataOnVMStarted     map[string]any

	exampleDataOnVMStoppedOnce sync.Once
	exampleDataOnVMStopped     map[string]any

	exampleDataOnVMRestartedOnce sync.Once
	exampleDataOnVMRestarted     map[string]any

	exampleDataOnVMDeallocatedOnce sync.Once
	exampleDataOnVMDeallocated     map[string]any

	exampleDataOnVMDeletedOnce sync.Once
	exampleDataOnVMDeleted     map[string]any
)

func (c *CreateVMComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateVMOnce, exampleOutputCreateVMBytes, &exampleOutputCreateVM)
}

func (c *StartVMComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputStartVMOnce, exampleOutputStartVMBytes, &exampleOutputStartVM)
}

func (c *StopVMComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputStopVMOnce, exampleOutputStopVMBytes, &exampleOutputStopVM)
}

func (c *RestartVMComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRestartVMOnce, exampleOutputRestartVMBytes, &exampleOutputRestartVM)
}

func (c *DeallocateVMComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeallocateVMOnce, exampleOutputDeallocateVMBytes, &exampleOutputDeallocateVM)
}

func (c *DeleteVMComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteVMOnce, exampleOutputDeleteVMBytes, &exampleOutputDeleteVM)
}

func (t *OnBlobCreated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBlobCreatedOnce, exampleDataOnBlobCreatedBytes, &exampleDataOnBlobCreated)
}

func (t *OnBlobDeleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBlobDeletedOnce, exampleDataOnBlobDeletedBytes, &exampleDataOnBlobDeleted)
}

func (t *OnImagePushed) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushedOnce, exampleDataOnImagePushedBytes, &exampleDataOnImagePushed)
}

func (t *OnImageDeleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImageDeletedOnce, exampleDataOnImageDeletedBytes, &exampleDataOnImageDeleted)
}

func (t *OnVMStarted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVMStartedOnce, exampleDataOnVMStartedBytes, &exampleDataOnVMStarted)
}

func (t *OnVMStopped) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVMStoppedOnce, exampleDataOnVMStoppedBytes, &exampleDataOnVMStopped)
}

func (t *OnVMRestarted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVMRestartedOnce, exampleDataOnVMRestartedBytes, &exampleDataOnVMRestarted)
}

func (t *OnVMDeallocated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVMDeallocatedOnce, exampleDataOnVMDeallocatedBytes, &exampleDataOnVMDeallocated)
}

func (t *OnVMDeleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVMDeletedOnce, exampleDataOnVMDeletedBytes, &exampleDataOnVMDeleted)
}
