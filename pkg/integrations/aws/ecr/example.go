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

//go:embed example_output_get_image.json
var exampleOutputGetImageBytes []byte

//go:embed example_output_get_image_scan_findings.json
var exampleOutputGetImageScanFindingsBytes []byte

var exampleDataOnImagePushOnce sync.Once
var exampleDataOnImagePush map[string]any

var exampleDataOnImageScanOnce sync.Once
var exampleDataOnImageScan map[string]any

var exampleOutputGetImageOnce sync.Once
var exampleOutputGetImage map[string]any

var exampleOutputGetImageScanFindingsOnce sync.Once
var exampleOutputGetImageScanFindings map[string]any

func (t *OnImagePush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushOnce, exampleDataOnImagePushBytes, &exampleDataOnImagePush)
}

func (t *OnImageScan) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImageScanOnce, exampleDataOnImageScanBytes, &exampleDataOnImageScan)
}

func (c *GetImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageOnce, exampleOutputGetImageBytes, &exampleOutputGetImage)
}

func (c *GetImageScanFindings) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageScanFindingsOnce, exampleOutputGetImageScanFindingsBytes, &exampleOutputGetImageScanFindings)
}
