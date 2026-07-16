package prometheus

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_workspace.json
var exampleOutputCreateWorkspaceBytes []byte
var exampleOutputCreateWorkspace = utils.NewEmbeddedJSON(exampleOutputCreateWorkspaceBytes)

//go:embed example_output_get_workspace.json
var exampleOutputGetWorkspaceBytes []byte
var exampleOutputGetWorkspace = utils.NewEmbeddedJSON(exampleOutputGetWorkspaceBytes)

//go:embed example_output_update_workspace.json
var exampleOutputUpdateWorkspaceBytes []byte
var exampleOutputUpdateWorkspace = utils.NewEmbeddedJSON(exampleOutputUpdateWorkspaceBytes)

//go:embed example_output_delete_workspace.json
var exampleOutputDeleteWorkspaceBytes []byte
var exampleOutputDeleteWorkspace = utils.NewEmbeddedJSON(exampleOutputDeleteWorkspaceBytes)

//go:embed example_output_query.json
var exampleOutputQueryBytes []byte
var exampleOutputQuery = utils.NewEmbeddedJSON(exampleOutputQueryBytes)

//go:embed example_output_query_range.json
var exampleOutputQueryRangeBytes []byte
var exampleOutputQueryRange = utils.NewEmbeddedJSON(exampleOutputQueryRangeBytes)

func (c *CreateWorkspace) ExampleOutput() map[string]any {
	return exampleOutputCreateWorkspace.Value()
}

func (c *GetWorkspace) ExampleOutput() map[string]any {
	return exampleOutputGetWorkspace.Value()
}

func (c *UpdateWorkspace) ExampleOutput() map[string]any {
	return exampleOutputUpdateWorkspace.Value()
}

func (c *DeleteWorkspace) ExampleOutput() map[string]any {
	return exampleOutputDeleteWorkspace.Value()
}

func (c *Query) ExampleOutput() map[string]any {
	return exampleOutputQuery.Value()
}

func (c *QueryRange) ExampleOutput() map[string]any {
	return exampleOutputQueryRange.Value()
}
