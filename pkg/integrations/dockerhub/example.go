package dockerhub

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_image_tag.json
var exampleOutputGetImageTagBytes []byte

//go:embed example_data_on_image_push.json
var exampleDataOnImagePushBytes []byte

//go:embed example_data_on_vulnerability_scan.json
var exampleDataOnVulnerabilityScanBytes []byte

var exampleOutputGetImageTagOnce sync.Once
var exampleOutputGetImageTag map[string]any

var exampleDataOnImagePushOnce sync.Once
var exampleDataOnImagePush map[string]any

var exampleDataOnVulnerabilityScanOnce sync.Once
var exampleDataOnVulnerabilityScan map[string]any

func getImageTagExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageTagOnce, exampleOutputGetImageTagBytes, &exampleOutputGetImageTag)
}

func onImagePushExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushOnce, exampleDataOnImagePushBytes, &exampleDataOnImagePush)
}

func onVulnerabilityScanExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVulnerabilityScanOnce, exampleDataOnVulnerabilityScanBytes, &exampleDataOnVulnerabilityScan)
}
