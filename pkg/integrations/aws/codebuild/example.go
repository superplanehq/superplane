package codebuild

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_build.json
var exampleDataOnBuildBytes []byte

//go:embed example_output_run_build.json
var exampleOutputRunBuildBytes []byte

var exampleDataOnBuildOnce sync.Once
var exampleDataOnBuild map[string]any

var exampleOutputRunBuildOnce sync.Once
var exampleOutputRunBuild map[string]any

func (t *OnBuild) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBuildOnce, exampleDataOnBuildBytes, &exampleDataOnBuild)
}

func (c *RunBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunBuildOnce, exampleOutputRunBuildBytes, &exampleOutputRunBuild)
}
