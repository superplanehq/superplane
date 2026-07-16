package jfrogartifactory

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_artifact_info.json
var exampleOutputGetArtifactInfoBytes []byte

//go:embed example_output_delete_artifact.json
var exampleOutputDeleteArtifactBytes []byte

//go:embed example_data_on_artifact_uploaded.json
var exampleDataOnArtifactUploadedBytes []byte
var exampleOutputGetArtifactInfo = utils.NewEmbeddedJSON(exampleOutputGetArtifactInfoBytes)
var exampleOutputDeleteArtifact = utils.NewEmbeddedJSON(exampleOutputDeleteArtifactBytes)
var exampleDataOnArtifactUploaded = utils.NewEmbeddedJSON(exampleDataOnArtifactUploadedBytes)

func (g *GetArtifactInfo) ExampleOutput() map[string]any {
	return exampleOutputGetArtifactInfo.Value()
}

func (d *DeleteArtifact) ExampleOutput() map[string]any {
	return exampleOutputDeleteArtifact.Value()
}

func (t *OnArtifactUploaded) ExampleData() map[string]any {
	return exampleDataOnArtifactUploaded.Value()
}
