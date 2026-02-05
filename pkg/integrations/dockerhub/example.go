package dockerhub

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_describe_image_tag.json
var exampleOutputDescribeImageTagBytes []byte

//go:embed example_data_on_image_push.json
var exampleDataOnImagePushBytes []byte

var exampleOutputDescribeImageTagOnce sync.Once
var exampleOutputDescribeImageTag map[string]any

var exampleDataOnImagePushOnce sync.Once
var exampleDataOnImagePush map[string]any

func (d *DescribeImageTag) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDescribeImageTagOnce, exampleOutputDescribeImageTagBytes, &exampleOutputDescribeImageTag)
}

func (t *OnImagePush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushOnce, exampleDataOnImagePushBytes, &exampleDataOnImagePush)
}
