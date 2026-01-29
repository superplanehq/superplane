package ecr

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_image_push.json
var exampleDataOnImagePushBytes []byte

//go:embed example_data_on_image_scan.json
var exampleDataOnImageScanBytes []byte

var exampleDataOnImagePushOnce sync.Once
var exampleDataOnImagePush map[string]any

var exampleDataOnImageScanOnce sync.Once
var exampleDataOnImageScan map[string]any

func (t *OnImagePush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushOnce, exampleDataOnImagePushBytes, &exampleDataOnImagePush)
}

func (t *OnImageScan) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImageScanOnce, exampleDataOnImageScanBytes, &exampleDataOnImageScan)
}
