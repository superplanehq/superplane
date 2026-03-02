package daytona

import (
	_ "embed"
	"sync"

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

var exampleOutputCreateSandboxOnce sync.Once
var exampleOutputCreateSandbox map[string]any

var exampleOutputCreateRepositorySandboxOnce sync.Once
var exampleOutputCreateRepositorySandbox map[string]any

var exampleOutputExecuteCodeOnce sync.Once
var exampleOutputExecuteCode map[string]any

var exampleOutputExecuteCommandOnce sync.Once
var exampleOutputExecuteCommand map[string]any

var exampleOutputGetPreviewURLOnce sync.Once
var exampleOutputGetPreviewURL map[string]any

var exampleOutputDeleteSandboxOnce sync.Once
var exampleOutputDeleteSandbox map[string]any

func (c *CreateSandbox) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSandboxOnce, exampleOutputCreateSandboxBytes, &exampleOutputCreateSandbox)
}

func (c *CreateRepositorySandbox) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateRepositorySandboxOnce,
		exampleOutputCreateRepositorySandboxBytes,
		&exampleOutputCreateRepositorySandbox,
	)
}

func (e *ExecuteCode) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputExecuteCodeOnce, exampleOutputExecuteCodeBytes, &exampleOutputExecuteCode)
}

func (e *ExecuteCommand) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputExecuteCommandOnce, exampleOutputExecuteCommandBytes, &exampleOutputExecuteCommand)
}

func (p *GetPreviewURLComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPreviewURLOnce, exampleOutputGetPreviewURLBytes, &exampleOutputGetPreviewURL)
}

func (d *DeleteSandbox) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteSandboxOnce, exampleOutputDeleteSandboxBytes, &exampleOutputDeleteSandbox)
}
