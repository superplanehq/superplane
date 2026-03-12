package artifactregistry

import (
	_ "embed"
	"sync"

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

var exampleOutputGetArtifactOnce sync.Once
var exampleOutputGetArtifact map[string]any

var exampleOutputGetArtifactAnalysisOnce sync.Once
var exampleOutputGetArtifactAnalysis map[string]any

var exampleDataOnArtifactPushOnce sync.Once
var exampleDataOnArtifactPush map[string]any

var exampleDataOnArtifactAnalysisOnce sync.Once
var exampleDataOnArtifactAnalysis map[string]any

func (c *GetArtifact) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetArtifactOnce, exampleOutputGetArtifactBytes, &exampleOutputGetArtifact)
}

func (c *GetArtifactAnalysis) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetArtifactAnalysisOnce, exampleOutputGetArtifactAnalysisBytes, &exampleOutputGetArtifactAnalysis)
}

func (t *OnArtifactPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnArtifactPushOnce, exampleDataOnArtifactPushBytes, &exampleDataOnArtifactPush)
}

func (t *OnArtifactAnalysis) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnArtifactAnalysisOnce, exampleDataOnArtifactAnalysisBytes, &exampleDataOnArtifactAnalysis)
}
