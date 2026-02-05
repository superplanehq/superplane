package dockerhub

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_list_tags.json
var exampleOutputListTagsBytes []byte

//go:embed example_data_on_image_pushed.json
var exampleDataOnImagePushedBytes []byte

var exampleOutputListTagsOnce sync.Once
var exampleOutputListTags map[string]any

var exampleDataOnImagePushedOnce sync.Once
var exampleDataOnImagePushed map[string]any

func (l *ListTags) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListTagsOnce, exampleOutputListTagsBytes, &exampleOutputListTags)
}

func (t *OnImagePushed) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushedOnce, exampleDataOnImagePushedBytes, &exampleDataOnImagePushed)
}
