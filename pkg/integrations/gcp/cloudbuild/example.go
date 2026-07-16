package cloudbuild

import (
	_ "embed"

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
var exampleOutputCreateBuild = utils.NewEmbeddedJSON(exampleOutputCreateBuildBytes)
var exampleOutputGetBuild = utils.NewEmbeddedJSON(exampleOutputGetBuildBytes)
var exampleDataOnBuildComplete = utils.NewEmbeddedJSON(exampleDataOnBuildCompleteBytes)
var exampleOutputRunTrigger = utils.NewEmbeddedJSON(exampleOutputRunTriggerBytes)

func (c *CreateBuild) ExampleOutput() map[string]any {
	return exampleOutputCreateBuild.Value()
}

func (c *GetBuild) ExampleOutput() map[string]any {
	return exampleOutputGetBuild.Value()
}

func (t *OnBuildComplete) ExampleData() map[string]any {
	return exampleDataOnBuildComplete.Value()
}

func (c *RunTrigger) ExampleOutput() map[string]any {
	return exampleOutputRunTrigger.Value()
}
