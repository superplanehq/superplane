package prometheus

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_workspace.json
var exampleOutputCreateWorkspaceBytes []byte

var exampleOutputCreateWorkspaceOnce sync.Once
var exampleOutputCreateWorkspace map[string]any

//go:embed example_output_get_workspace.json
var exampleOutputGetWorkspaceBytes []byte

var exampleOutputGetWorkspaceOnce sync.Once
var exampleOutputGetWorkspace map[string]any

//go:embed example_output_update_workspace.json
var exampleOutputUpdateWorkspaceBytes []byte

var exampleOutputUpdateWorkspaceOnce sync.Once
var exampleOutputUpdateWorkspace map[string]any

//go:embed example_output_delete_workspace.json
var exampleOutputDeleteWorkspaceBytes []byte

var exampleOutputDeleteWorkspaceOnce sync.Once
var exampleOutputDeleteWorkspace map[string]any

//go:embed example_output_query.json
var exampleOutputQueryBytes []byte

var exampleOutputQueryOnce sync.Once
var exampleOutputQuery map[string]any

//go:embed example_output_query_range.json
var exampleOutputQueryRangeBytes []byte

var exampleOutputQueryRangeOnce sync.Once
var exampleOutputQueryRange map[string]any

func (c *CreateWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateWorkspaceOnce,
		exampleOutputCreateWorkspaceBytes,
		&exampleOutputCreateWorkspace,
	)
}

func (c *GetWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetWorkspaceOnce,
		exampleOutputGetWorkspaceBytes,
		&exampleOutputGetWorkspace,
	)
}

func (c *UpdateWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateWorkspaceOnce,
		exampleOutputUpdateWorkspaceBytes,
		&exampleOutputUpdateWorkspace,
	)
}

func (c *DeleteWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteWorkspaceOnce,
		exampleOutputDeleteWorkspaceBytes,
		&exampleOutputDeleteWorkspace,
	)
}

func (c *Query) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputQueryOnce,
		exampleOutputQueryBytes,
		&exampleOutputQuery,
	)
}

func (c *QueryRange) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputQueryRangeOnce,
		exampleOutputQueryRangeBytes,
		&exampleOutputQueryRange,
	)
}
