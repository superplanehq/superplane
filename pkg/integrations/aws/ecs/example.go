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

var exampleOutputDescribeServiceOnce sync.Once
var exampleOutputDescribeService map[string]any

var exampleOutputRunTaskOnce sync.Once
var exampleOutputRunTask map[string]any

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
