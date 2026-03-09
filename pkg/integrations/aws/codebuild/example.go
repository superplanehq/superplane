package codebuild

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_build.json
var exampleDataOnBuildBytes []byte

var exampleDataOnBuildOnce sync.Once
var exampleDataOnBuild map[string]any

func (t *OnBuild) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnBuildOnce,
		exampleDataOnBuildBytes,
		&exampleDataOnBuild,
	)
}

//go:embed example_output_start_build.json
var exampleOutputStartBuildBytes []byte

var exampleOutputStartBuildOnce sync.Once
var exampleOutputStartBuild map[string]any

func (s *StartBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputStartBuildOnce,
		exampleOutputStartBuildBytes,
		&exampleOutputStartBuild,
	)
}

//go:embed example_output_get_build_status.json
var exampleOutputGetBuildStatusBytes []byte

var exampleOutputGetBuildStatusOnce sync.Once
var exampleOutputGetBuildStatus map[string]any

func (c *GetBuildStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetBuildStatusOnce,
		exampleOutputGetBuildStatusBytes,
		&exampleOutputGetBuildStatus,
	)
}

//go:embed example_output_stop_build.json
var exampleOutputStopBuildBytes []byte

var exampleOutputStopBuildOnce sync.Once
var exampleOutputStopBuild map[string]any

func (c *StopBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputStopBuildOnce,
		exampleOutputStopBuildBytes,
		&exampleOutputStopBuild,
	)
}
