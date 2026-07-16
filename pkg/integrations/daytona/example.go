package daytona

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_sandbox.json
var exampleOutputCreateSandboxBytes []byte

//go:embed example_output_create_repository_sandbox.json
var exampleOutputCreateRepositorySandboxBytes []byte

//go:embed example_output_execute_code.json
var exampleOutputExecuteCodeBytes []byte

//go:embed example_output_execute_command.json
var exampleOutputExecuteCommandBytes []byte

//go:embed example_output_get_preview_url.json
var exampleOutputGetPreviewURLBytes []byte

//go:embed example_output_delete_sandbox.json
var exampleOutputDeleteSandboxBytes []byte
var exampleOutputCreateSandbox = utils.NewEmbeddedJSON(exampleOutputCreateSandboxBytes)
var exampleOutputCreateRepositorySandbox = utils.NewEmbeddedJSON(exampleOutputCreateRepositorySandboxBytes)
var exampleOutputExecuteCode = utils.NewEmbeddedJSON(exampleOutputExecuteCodeBytes)
var exampleOutputExecuteCommand = utils.NewEmbeddedJSON(exampleOutputExecuteCommandBytes)
var exampleOutputGetPreviewURL = utils.NewEmbeddedJSON(exampleOutputGetPreviewURLBytes)
var exampleOutputDeleteSandbox = utils.NewEmbeddedJSON(exampleOutputDeleteSandboxBytes)

func (c *CreateSandbox) ExampleOutput() map[string]any {
	return exampleOutputCreateSandbox.Value()
}

func (c *CreateRepositorySandbox) ExampleOutput() map[string]any {
	return exampleOutputCreateRepositorySandbox.Value()
}

func (e *ExecuteCode) ExampleOutput() map[string]any {
	return exampleOutputExecuteCode.Value()
}

func (e *ExecuteCommand) ExampleOutput() map[string]any {
	return exampleOutputExecuteCommand.Value()
}

func (p *GetPreviewURLComponent) ExampleOutput() map[string]any {
	return exampleOutputGetPreviewURL.Value()
}

func (d *DeleteSandbox) ExampleOutput() map[string]any {
	return exampleOutputDeleteSandbox.Value()
}
