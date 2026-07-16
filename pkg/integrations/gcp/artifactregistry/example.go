package artifactregistry

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_artifact.json
var exampleOutputGetArtifactBytes []byte

//go:embed example_output_get_artifact_analysis.json
var exampleOutputGetArtifactAnalysisBytes []byte

//go:embed example_data_on_artifact_push.json
var exampleDataOnArtifactPushBytes []byte

//go:embed example_data_on_artifact_analysis.json
var exampleDataOnArtifactAnalysisBytes []byte
var exampleOutputGetArtifact = utils.NewEmbeddedJSON(exampleOutputGetArtifactBytes)
var exampleOutputGetArtifactAnalysis = utils.NewEmbeddedJSON(exampleOutputGetArtifactAnalysisBytes)
var exampleDataOnArtifactPush = utils.NewEmbeddedJSON(exampleDataOnArtifactPushBytes)
var exampleDataOnArtifactAnalysis = utils.NewEmbeddedJSON(exampleDataOnArtifactAnalysisBytes)

func (c *GetArtifact) ExampleOutput() map[string]any {
	return exampleOutputGetArtifact.Value()
}

func (c *GetArtifactAnalysis) ExampleOutput() map[string]any {
	return exampleOutputGetArtifactAnalysis.Value()
}

func (t *OnArtifactPush) ExampleData() map[string]any {
	return exampleDataOnArtifactPush.Value()
}

func (t *OnArtifactAnalysis) ExampleData() map[string]any {
	return exampleDataOnArtifactAnalysis.Value()
}
