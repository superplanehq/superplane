package dockerhub

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_image_tag.json
var exampleOutputGetImageTagBytes []byte

//go:embed example_data_on_image_push.json
var exampleDataOnImagePushBytes []byte
var exampleOutputGetImageTag = utils.NewEmbeddedJSON(exampleOutputGetImageTagBytes)
var exampleDataOnImagePush = utils.NewEmbeddedJSON(exampleDataOnImagePushBytes)

func getImageTagExampleOutput() map[string]any {
	return exampleOutputGetImageTag.Value()
}

func onImagePushExampleData() map[string]any {
	return exampleDataOnImagePush.Value()
}
