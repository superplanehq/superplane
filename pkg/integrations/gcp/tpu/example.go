package tpu

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_node.json
var exampleOutputCreateNodeBytes []byte

//go:embed example_output_get_node.json
var exampleOutputGetNodeBytes []byte

//go:embed example_output_delete_node.json
var exampleOutputDeleteNodeBytes []byte

var (
	exampleOutputCreateNodeOnce sync.Once
	exampleOutputCreateNode     map[string]any

	exampleOutputGetNodeOnce sync.Once
	exampleOutputGetNode     map[string]any

	exampleOutputDeleteNodeOnce sync.Once
	exampleOutputDeleteNode     map[string]any
)

func (c *CreateNode) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateNodeOnce, exampleOutputCreateNodeBytes, &exampleOutputCreateNode)
}

func (c *GetNode) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetNodeOnce, exampleOutputGetNodeBytes, &exampleOutputGetNode)
}

func (c *DeleteNode) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteNodeOnce, exampleOutputDeleteNodeBytes, &exampleOutputDeleteNode)
}
