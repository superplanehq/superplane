package ecs

import (
	_ "embed"
	"sync"

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

var exampleOutputDescribeServiceOnce sync.Once
var exampleOutputDescribeService map[string]any

var exampleOutputRunTaskOnce sync.Once
var exampleOutputRunTask map[string]any

var exampleOutputStopTaskOnce sync.Once
var exampleOutputStopTask map[string]any

var exampleOutputCreateServiceOnce sync.Once
var exampleOutputCreateService map[string]any

var exampleOutputUpdateServiceOnce sync.Once
var exampleOutputUpdateService map[string]any

var exampleOutputExecuteCommandOnce sync.Once
var exampleOutputExecuteCommand map[string]any

func (c *DescribeService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDescribeServiceOnce,
		exampleOutputDescribeServiceBytes,
		&exampleOutputDescribeService,
	)
}

func (c *RunTask) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRunTaskOnce,
		exampleOutputRunTaskBytes,
		&exampleOutputRunTask,
	)
}

func (c *StopTask) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputStopTaskOnce,
		exampleOutputStopTaskBytes,
		&exampleOutputStopTask,
	)
}

func (c *CreateService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateServiceOnce,
		exampleOutputCreateServiceBytes,
		&exampleOutputCreateService,
	)
}

func (c *UpdateService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateServiceOnce,
		exampleOutputUpdateServiceBytes,
		&exampleOutputUpdateService,
	)
}

func (c *ExecuteCommand) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputExecuteCommandOnce,
		exampleOutputExecuteCommandBytes,
		&exampleOutputExecuteCommand,
	)
}
