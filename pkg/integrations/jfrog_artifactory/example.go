package jfrogartifactory

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_artifact_info.json
var exampleOutputGetArtifactInfoBytes []byte

//go:embed example_output_delete_artifact.json
var exampleOutputDeleteArtifactBytes []byte

//go:embed example_data_on_artifact_uploaded.json
var exampleDataOnArtifactUploadedBytes []byte

var exampleOutputGetArtifactInfoOnce sync.Once
var exampleOutputGetArtifactInfo map[string]any

var exampleOutputDeleteArtifactOnce sync.Once
var exampleOutputDeleteArtifact map[string]any

var exampleDataOnArtifactUploadedOnce sync.Once
var exampleDataOnArtifactUploaded map[string]any

func (g *GetArtifactInfo) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetArtifactInfoOnce, exampleOutputGetArtifactInfoBytes, &exampleOutputGetArtifactInfo)
}

func (d *DeleteArtifact) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteArtifactOnce, exampleOutputDeleteArtifactBytes, &exampleOutputDeleteArtifact)
}

func (t *OnArtifactUploaded) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnArtifactUploadedOnce, exampleDataOnArtifactUploadedBytes, &exampleDataOnArtifactUploaded)
}
