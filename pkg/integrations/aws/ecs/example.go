package ecs

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_describe_service.json
var exampleOutputDescribeServiceBytes []byte

//go:embed example_output_run_task.json
var exampleOutputRunTaskBytes []byte

//go:embed example_output_stop_task.json
var exampleOutputStopTaskBytes []byte

//go:embed example_output_create_service.json
var exampleOutputCreateServiceBytes []byte

//go:embed example_output_update_service.json
var exampleOutputUpdateServiceBytes []byte

//go:embed example_output_execute_command.json
var exampleOutputExecuteCommandBytes []byte
var exampleOutputDescribeService = utils.NewEmbeddedJSON(exampleOutputDescribeServiceBytes)
var exampleOutputRunTask = utils.NewEmbeddedJSON(exampleOutputRunTaskBytes)
var exampleOutputStopTask = utils.NewEmbeddedJSON(exampleOutputStopTaskBytes)
var exampleOutputCreateService = utils.NewEmbeddedJSON(exampleOutputCreateServiceBytes)
var exampleOutputUpdateService = utils.NewEmbeddedJSON(exampleOutputUpdateServiceBytes)
var exampleOutputExecuteCommand = utils.NewEmbeddedJSON(exampleOutputExecuteCommandBytes)

func (c *DescribeService) ExampleOutput() map[string]any {
	return exampleOutputDescribeService.Value()
}

func (c *RunTask) ExampleOutput() map[string]any {
	return exampleOutputRunTask.Value()
}

func (c *StopTask) ExampleOutput() map[string]any {
	return exampleOutputStopTask.Value()
}

func (c *CreateService) ExampleOutput() map[string]any {
	return exampleOutputCreateService.Value()
}

func (c *UpdateService) ExampleOutput() map[string]any {
	return exampleOutputUpdateService.Value()
}

func (c *ExecuteCommand) ExampleOutput() map[string]any {
	return exampleOutputExecuteCommand.Value()
}
