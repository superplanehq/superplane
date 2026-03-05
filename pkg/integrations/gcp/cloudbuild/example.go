package cloudbuild

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_build.json
var exampleOutputCreateBuildBytes []byte

//go:embed example_output_get_build.json
var exampleOutputGetBuildBytes []byte

//go:embed example_data_on_build_complete.json
var exampleDataOnBuildCompleteBytes []byte

//go:embed example_output_run_trigger.json
var exampleOutputRunTriggerBytes []byte

var exampleOutputCreateBuildOnce sync.Once
var exampleOutputCreateBuild map[string]any

var exampleOutputGetBuildOnce sync.Once
var exampleOutputGetBuild map[string]any

var exampleDataOnBuildCompleteOnce sync.Once
var exampleDataOnBuildComplete map[string]any

var exampleOutputRunTriggerOnce sync.Once
var exampleOutputRunTrigger map[string]any

func (c *CreateBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateBuildOnce, exampleOutputCreateBuildBytes, &exampleOutputCreateBuild)
}

func (c *GetBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetBuildOnce, exampleOutputGetBuildBytes, &exampleOutputGetBuild)
}

func (t *OnBuildComplete) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBuildCompleteOnce, exampleDataOnBuildCompleteBytes, &exampleDataOnBuildComplete)
}

func (c *RunTrigger) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunTriggerOnce, exampleOutputRunTriggerBytes, &exampleOutputRunTrigger)
}
