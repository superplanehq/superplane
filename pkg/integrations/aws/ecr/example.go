package ecr

import (
	_ "embed"

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

//go:embed example_output_scan_image.json
var exampleOutputScanImageBytes []byte
var exampleDataOnImagePush = utils.NewEmbeddedJSON(exampleDataOnImagePushBytes)
var exampleDataOnImageScan = utils.NewEmbeddedJSON(exampleDataOnImageScanBytes)
var exampleOutputGetImage = utils.NewEmbeddedJSON(exampleOutputGetImageBytes)
var exampleOutputGetImageScanFindings = utils.NewEmbeddedJSON(exampleOutputGetImageScanFindingsBytes)
var exampleOutputScanImage = utils.NewEmbeddedJSON(exampleOutputScanImageBytes)

func (t *OnImagePush) ExampleData() map[string]any {
	return exampleDataOnImagePush.Value()
}

func (t *OnImageScan) ExampleData() map[string]any {
	return exampleDataOnImageScan.Value()
}

func (c *GetImage) ExampleOutput() map[string]any {
	return exampleOutputGetImage.Value()
}

func (c *GetImageScanFindings) ExampleOutput() map[string]any {
	return exampleOutputGetImageScanFindings.Value()
}

func (c *ScanImage) ExampleOutput() map[string]any {
	return exampleOutputScanImage.Value()
}
