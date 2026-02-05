package compute

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_virtual_machine.json
var exampleOutputCreateVMBytes []byte

var exampleOutputCreateVMOnce sync.Once
var exampleOutputCreateVM map[string]any

func (c *CreateVirtualMachine) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateVMOnce, exampleOutputCreateVMBytes, &exampleOutputCreateVM)
}
